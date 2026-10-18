package engine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"reflect"
	"strings"
	"testing"
)

func encodingFixture() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(11, -3, 51, 17))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			img.SetNRGBA(x+11, y-3, color.NRGBA{uint8(x * 7), uint8(y * 13), uint8(x*11 + y*17), uint8(((x + y) % 4) * 85)})
		}
	}
	return img
}

func TestEncodeContextPreservesBytes(t *testing.T) {
	// Captured before adding context support, on the same nonzero-origin image
	// with alpha and enough visible colors to require GIF palette extraction.
	want := map[string]string{
		"png":   "e9f1f9423fe1e67cdcde22d0f781692c1c765623e5cb5a0da79c0be28f1f6413",
		"jpeg":  "85f8c818a6923ea95af2d887c4595bad02b59176a83b72da6c354ca7cb0dc89b",
		"gif":   "1be274c845cde6fc452b032b9eed5d54744387c37fda243a1205a82a1575f36d",
		"svg":   "fd6cc2826eecdd705ab7f86d7fb46c1e0649c036f53aa4e172a409dc3fc52cb2",
		"pbm":   "7b8e2a7472e1af668bbb65b9e7bc217bef943e938c15cc2ee52867b44864f4b0",
		"ascii": "1816780d139cb2914c40087eb288d380ad06e756950a4924c7fc5eb5f5c349cb",
	}
	for _, format := range Formats() {
		t.Run(format, func(t *testing.T) {
			var old, current bytes.Buffer
			if err := Encode(format, &old, encodingFixture()); err != nil {
				t.Fatal(err)
			}
			if err := EncodeContext(context.Background(), format, &current, encodingFixture()); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(old.Bytes(), current.Bytes()) || fmt.Sprintf("%x", sha256.Sum256(current.Bytes())) != want[format] {
				t.Fatal("successful encoder bytes changed")
			}
		})
	}
	old, err := Palettize(encodingFixture())
	if err != nil {
		t.Fatal(err)
	}
	current, err := PalettizeContext(context.Background(), encodingFixture())
	if err != nil || !reflect.DeepEqual(old, current) {
		t.Fatalf("palettization compatibility changed: %v", err)
	}
}

type encodeCancelImage struct {
	image.Image
	cancel context.CancelFunc
	atCall int
	calls  int
}

func (img *encodeCancelImage) At(x, y int) color.Color {
	img.calls++
	if img.calls == img.atCall {
		img.cancel()
	}
	return img.Image.At(x, y)
}

func TestPalettizeContextCancellationPhases(t *testing.T) {
	small := image.NewNRGBA(image.Rect(3, -2, 11, 2))
	many := image.NewNRGBA(image.Rect(3, -2, 35, 14))
	for y := many.Rect.Min.Y; y < many.Rect.Max.Y; y++ {
		for x := many.Rect.Min.X; x < many.Rect.Max.X; x++ {
			many.SetNRGBA(x, y, color.NRGBA{uint8((x - 3) * 8), uint8((y + 2) * 16), 71, 255})
		}
	}
	for _, tt := range []struct {
		name       string
		img        image.Image
		atCall     int
		maxSamples int
	}{
		{"scan", small, 1, 8},
		{"last scan row", small, 32, 32},
		{"mapping", small, 33, 40},
		{"last mapping row", small, 64, 64},
		{"overflow extraction", many, 513, 544},
		{"last extraction row", many, 1024, 1024},
		{"overflow mapping", many, 1025, 1056},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			img := &encodeCancelImage{Image: tt.img, cancel: cancel, atCall: tt.atCall}
			out, err := PalettizeContext(ctx, img)
			if !errors.Is(err, context.Canceled) || out != nil {
				t.Fatalf("canceled phase returned image=%v error=%v", out != nil, err)
			}
			if img.calls < tt.atCall || img.calls > tt.maxSamples {
				t.Fatalf("did not stop at row boundary: %d reads, want %d..%d", img.calls, tt.atCall, tt.maxSamples)
			}
		})
	}
}

func TestEncodeContextStopsHandwrittenRows(t *testing.T) {
	for _, format := range []string{"jpeg", "gif", "svg", "pbm", "ascii"} {
		for _, atCall := range []int{1, 32} {
			t.Run(fmt.Sprintf("%s/at=%d", format, atCall), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				img := &encodeCancelImage{Image: image.NewNRGBA(image.Rect(4, -2, 12, 2)), cancel: cancel, atCall: atCall}
				var output bytes.Buffer
				if err := EncodeContext(ctx, format, &output, img); !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation lost: %v", err)
				}
				limit := 8
				if atCall == 32 {
					limit = 32
				}
				if img.calls > limit || img.calls < atCall {
					t.Fatalf("handwritten pass ignored cancellation: %d reads", img.calls)
				}
			})
		}
	}
}

type encodeCancelWriter struct {
	cancel context.CancelFunc
	calls  int
	err    error
}

func (w *encodeCancelWriter) Write(p []byte) (int, error) {
	w.calls++
	w.cancel()
	if w.err != nil {
		return 0, w.err
	}
	return len(p), nil
}

func TestEncodeContextWriterCancellationAndErrors(t *testing.T) {
	sentinel := errors.New("writer failed")
	for _, format := range Formats() {
		for _, writerErr := range []error{nil, sentinel} {
			t.Run(fmt.Sprintf("%s/error=%v", format, writerErr), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				writer := &encodeCancelWriter{cancel: cancel, err: writerErr}
				want := writerErr
				if want == nil {
					want = context.Canceled
				}
				if err := EncodeContext(ctx, format, writer, encodingFixture()); !errors.Is(err, want) {
					t.Fatalf("got %v, want %v", err, want)
				}
				if writer.calls != 1 {
					t.Fatalf("wrote after cancellation: %d writes", writer.calls)
				}
			})
		}
	}
}

func TestEncodeContextRejectsCanceledOrNilContextBeforeReading(t *testing.T) {
	for _, format := range Formats() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		img := &encodeCancelImage{Image: encodingFixture(), cancel: cancel}
		var out bytes.Buffer
		if err := EncodeContext(ctx, format, &out, img); !errors.Is(err, context.Canceled) || out.Len() != 0 || img.calls != 0 {
			t.Fatalf("%s did work on canceled context: %v", format, err)
		}
		if err := EncodeContext(nil, format, io.Discard, img); err == nil || !strings.Contains(err.Error(), "context is nil") {
			t.Fatalf("nil context: %v", err)
		}
	}
	if _, err := PalettizeContext(nil, encodingFixture()); err == nil || !strings.Contains(err.Error(), "context is nil") {
		t.Fatalf("nil palettize context: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	img := &encodeCancelImage{Image: encodingFixture(), cancel: cancel}
	if result, err := PalettizeContext(ctx, img); !errors.Is(err, context.Canceled) || result != nil || img.calls != 0 {
		t.Fatalf("palettize did work on canceled context: %v", err)
	}
}

func TestEncodeContextPNGObservesCancellationDuringRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	img := &encodeCancelImage{Image: encodingFixture(), cancel: cancel, atCall: 1}
	if err := EncodeContext(ctx, "png", io.Discard, img); !errors.Is(err, context.Canceled) {
		t.Fatalf("PNG lost cancellation during a codec step: %v", err)
	}
}
