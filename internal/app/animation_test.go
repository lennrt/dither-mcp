package app

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func gifFixture(t *testing.T, transparent bool, disposals []byte) ([]byte, *gif.GIF) {
	t.Helper()
	palette := color.Palette{color.NRGBA{0, 0, 255, 255}, color.NRGBA{255, 0, 0, 255}, color.NRGBA{0, 255, 0, 255}, color.NRGBA{255, 255, 255, 255}}
	if transparent {
		palette[0] = color.NRGBA{}
	}
	frames := []*image.Paletted{image.NewPaletted(image.Rect(0, 0, 1, 1), palette), image.NewPaletted(image.Rect(0, 0, 1, 1), palette), image.NewPaletted(image.Rect(2, 0, 3, 1), palette)}
	frames[0].Pix[0] = 1
	frames[1].Pix[0] = 2
	frames[2].Pix[0] = 3
	original := &gif.GIF{Image: frames, Delay: []int{3, 7, 11}, Disposal: disposals, LoopCount: 2, BackgroundIndex: 0, Config: image.Config{ColorModel: palette, Width: 3, Height: 1}}
	var encoded bytes.Buffer
	if err := gif.EncodeAll(&encoded, original); err != nil {
		t.Fatal(err)
	}
	decoded, err := boundedGIF(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes(), decoded
}

func TestGIFCompositesLogicalBackgroundAndPrevious(t *testing.T) {
	for _, transparent := range []bool{false, true} {
		_, g := gifFixture(t, transparent, []byte{gif.DisposalNone, gif.DisposalPrevious, gif.DisposalNone})
		frames, err := compositeGIF(context.Background(), g)
		if err != nil {
			t.Fatal(err)
		}
		wantBackground := color.NRGBA{0, 0, 255, 255}
		if transparent {
			wantBackground = color.NRGBA{}
		}
		if got := color.NRGBAModel.Convert(frames[0].At(1, 0)).(color.NRGBA); got != wantBackground {
			t.Fatalf("transparent=%v: initial background %v, want %v", transparent, got, wantBackground)
		}
		if got := color.NRGBAModel.Convert(frames[2].At(0, 0)).(color.NRGBA); got != (color.NRGBA{255, 0, 0, 255}) {
			t.Fatalf("previous disposal lost prior red pixel: %v", got)
		}
		if got := color.NRGBAModel.Convert(frames[1].At(0, 0)).(color.NRGBA); got != (color.NRGBA{0, 255, 0, 255}) {
			t.Fatalf("stored snapshot mutated during disposal: %v", got)
		}
	}
}

func TestGIFDisposalBackgroundRestoresColorOrTransparency(t *testing.T) {
	for _, transparent := range []bool{false, true} {
		_, g := gifFixture(t, transparent, []byte{gif.DisposalNone, gif.DisposalBackground, gif.DisposalNone})
		frames, err := compositeGIF(context.Background(), g)
		if err != nil {
			t.Fatal(err)
		}
		want := color.NRGBA{0, 0, 255, 255}
		if transparent {
			want = color.NRGBA{}
		}
		if got := color.NRGBAModel.Convert(frames[2].At(0, 0)).(color.NRGBA); got != want {
			t.Fatalf("transparent=%v: restored %v, want %v", transparent, got, want)
		}
	}
}

func TestGIFSourceAnimationPreservesTimingAndLoop(t *testing.T) {
	s, root := securityFixture(t)
	data, _ := gifFixture(t, false, []byte{gif.DisposalNone, gif.DisposalBackground, gif.DisposalNone})
	if err := os.WriteFile(filepath.Join(root, "source.gif"), data, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := securityCall(s, "dither_animate", AnimateRequest{RenderRequest: RenderRequest{Input: "source.gif", Output: "result.gif", Options: engine.Config{Width: 3, Height: 1}}, Effect: "source"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "result.gif"))
	if err != nil {
		t.Fatal(err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Image) != 3 || g.LoopCount != 2 {
		t.Fatalf("frames=%d loop=%d, want 3,2", len(g.Image), g.LoopCount)
	}
	for i, want := range []int{3, 7, 11} {
		if g.Delay[i] != want {
			t.Fatalf("delay[%d]=%d want %d", i, g.Delay[i], want)
		}
	}
}

func TestAnimationPreflightRejectsAggregateBeforeAllocation(t *testing.T) {
	if _, _, err := frameDimensions(image.Rect(0, 0, 8, 8), engine.Config{Width: 4096, Height: 4096}, 5); err == nil {
		t.Fatal("five 16MP frames passed 64MP budget")
	}
	if _, _, err := frameDimensions(image.Rect(0, 0, 8, 8), engine.Config{Width: 4096, Height: 4096}, 4); err != nil {
		t.Fatalf("exact budget rejected: %v", err)
	}
	if _, _, err := frameDimensions(image.Rect(0, 0, 8, 4), engine.Config{Crop: &engine.Rect{X: 7, Width: 2, Height: 1}}, 1); err == nil {
		t.Fatal("crop outside source accepted")
	}
	if err := validateSpriteDimensions(4096, 4096, 4, 4); err == nil {
		t.Fatal("oversized spritesheet accepted")
	}
}

func TestAnimationPreflightMatchesAspectAndCrop(t *testing.T) {
	im := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	for _, c := range []engine.Config{{}, {Width: 3}, {Height: 3}, {Width: 3, Height: 7}, {Crop: &engine.Rect{X: 2, Y: 1, Width: 4, Height: 4}, Width: 3, PixelScale: 2}} {
		w, h, err := frameDimensions(im.Bounds(), c, 1)
		if err != nil {
			t.Fatal(err)
		}
		out, err := engine.Process(context.Background(), im, c)
		if err != nil {
			t.Fatal(err)
		}
		if out.Bounds().Dx() != w || out.Bounds().Dy() != h {
			t.Fatalf("preflight=%dx%d actual=%v config=%+v", w, h, out.Bounds(), c)
		}
	}
}

func TestGIFCompositionCancellation(t *testing.T) {
	_, g := gifFixture(t, false, []byte{0, 0, 0})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := compositeGIF(ctx, g); err == nil {
		t.Fatal("cancelled composition succeeded")
	}
}

func TestGIFFrameRateRounding(t *testing.T) {
	svc, err := New(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	frames := make([]*image.NRGBA, 24)
	for i := range frames {
		frames[i] = image.NewNRGBA(image.Rect(0, 0, 2, 2))
	}
	a, err := svc.publishFrames(context.Background(), "loop.gif", "gif", frames, nil, 12, 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := svc.read(a.Path, MaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	sum := 0
	for _, d := range g.Delay {
		if d < 8 || d > 9 {
			t.Fatalf("unexpected frame delay %d", d)
		}
		sum += d
	}
	if sum != 200 {
		t.Fatalf("duration=%dcs, want200", sum)
	}
}
