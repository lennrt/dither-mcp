package engine

import (
	"bytes"
	"context"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func TestInterchangeReferencePixels(t *testing.T) {
	src := image.NewNRGBA(image.Rect(5, -3, 8, -1))
	colors := []color.NRGBA{{255, 0, 0, 255}, {19, 87, 231, 128}, {99, 33, 127, 0}, {2, 3, 4, 1}, {7, 8, 9, 254}, {31, 63, 127, 255}}
	for i, c := range colors {
		src.SetNRGBA(5+i%3, -3+i/3, c)
	}
	for _, format := range []string{"bmp", "tiff", "tif"} {
		t.Run(format, func(t *testing.T) {
			var b bytes.Buffer
			if err := EncodeContext(context.Background(), format, &b, src); err != nil {
				t.Fatal(err)
			}
			var decoded image.Image
			var err error
			if format == "bmp" {
				decoded, err = bmp.Decode(&b)
			} else {
				decoded, err = tiff.Decode(&b)
			}
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Bounds() != image.Rect(0, 0, 3, 2) {
				t.Fatal(decoded.Bounds())
			}
			matte := image.NewRGBA(image.Rect(0, 0, 3, 2))
			draw.Draw(matte, matte.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
			draw.Draw(matte, matte.Bounds(), src, src.Bounds().Min, draw.Over)
			for i, want := range colors {
				if format == "bmp" {
					want = color.NRGBAModel.Convert(matte.At(i%3, i/3)).(color.NRGBA)
				}
				got := color.NRGBAModel.Convert(decoded.At(i%3, i/3)).(color.NRGBA)
				if got != want {
					t.Fatalf("pixel %d=%v want %v", i, got, want)
				}
			}
		})
	}
	if FormatFromExtension(".TIF") != "tiff" {
		t.Fatal("tif alias missing")
	}
}
