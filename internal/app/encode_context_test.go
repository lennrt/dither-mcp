package app

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lennrt/dither-mcp/engine"
)

type cancelEncodingImage struct {
	image.Image
	cancel context.CancelFunc
	calls  int
}

func (img *cancelEncodingImage) At(x, y int) color.Color {
	img.calls++
	img.cancel()
	return img.Image.At(x, y)
}

func TestBoundedEncodingPropagatesContextAndDiscardsPartialBytes(t *testing.T) {
	for _, format := range engine.Formats() {
		t.Run(format, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			img := &cancelEncodingImage{Image: image.NewNRGBA(image.Rect(0, 0, 8, 4)), cancel: cancel}
			data, err := encodeBounded(ctx, format, img, 0)
			if !errors.Is(err, context.Canceled) || data != nil {
				t.Fatalf("canceled encoder returned data=%d, error=%v", len(data), err)
			}
			if img.calls == 0 || (format != "png" && img.calls > 8) {
				t.Fatalf("encoder did not stop at its next row boundary: %d samples", img.calls)
			}
		})
	}
}

type cancelGIFColor struct{ cancel context.CancelFunc }

func (c cancelGIFColor) RGBA() (uint32, uint32, uint32, uint32) {
	c.cancel()
	return 0, 0, 0, 65535
}

func TestAnimatedGIFWriterObservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	frame := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{cancelGIFColor{cancel}, color.White})
	buffer := &boundedBuffer{max: int(MaxBytes), ctx: ctx}
	if err := gif.EncodeAll(buffer, &gif.GIF{Image: []*image.Paletted{frame}, Delay: []int{1}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("animation encoder lost cancellation: %v", err)
	}
	length := buffer.Len()
	if n, err := buffer.Write([]byte("late")); n != 0 || !errors.Is(err, context.Canceled) || buffer.Len() != length {
		t.Fatal("canceled final-encoding buffer accepted more output")
	}
}

func TestPreCanceledExportWorkflowsDoNotPublishOrHoldAdmission(t *testing.T) {
	s, root := securityFixture(t)
	request := RenderRequest{Input: "input.png", Output: "canceled.png", Options: engine.Config{Width: 4}}
	for _, operation := range []struct {
		name    string
		request any
	}{
		{"dither_render", request},
		{"dither_compare", CompareRequest{RenderRequest: request, Palettes: []string{"mono", "gameboy"}}},
		{"dither_studio", StudioRequest{Input: "input.png"}},
		{"dither_preview", PreviewRequest{Input: "input.png"}},
		{"dither_animate", AnimateRequest{RenderRequest: request, Frames: 2}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			raw, _ := json.Marshal(operation.request)
			if _, err := s.Do(ctx, operation.name, raw); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled operation returned %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, "canceled.png")); !os.IsNotExist(err) {
				t.Fatal("canceled workflow published output")
			}
			fresh, done := context.WithTimeout(context.Background(), time.Second)
			defer done()
			if _, err := s.Do(fresh, "dither_catalog", []byte("{}")); err != nil {
				t.Fatalf("canceled export retained service admission: %v", err)
			}
		})
	}
}
