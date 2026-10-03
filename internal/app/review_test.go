package app

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestReviewedGIFAndExtractionCases(t *testing.T) {
	root := t.TempDir()
	svc, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	p := color.Palette{color.RGBA{240, 220, 200, 255}, color.Black}
	frame := image.NewPaletted(image.Rect(2, 1, 4, 3), p)
	for i := range frame.Pix {
		frame.Pix[i] = 1
	}
	g := &gif.GIF{Image: []*image.Paletted{frame}, Delay: []int{8}, Config: image.Config{ColorModel: p, Width: 6, Height: 4}}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "partial.gif"), b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	value, err := svc.Do(context.Background(), "dither_inspect", []byte(`{"input":"partial.gif"}`))
	if err != nil {
		t.Fatal(err)
	}
	meta := value.(Inspection)
	if meta.Width != 6 || meta.Height != 4 || meta.HasAlpha {
		t.Fatalf("wrong logical-screen metadata: %+v", meta)
	}
	im, _, _, err := svc.decode(context.Background(), "partial.gif")
	if err != nil {
		t.Fatal(err)
	}
	r, gc, bc, _ := im.At(0, 0).RGBA()
	if r != 240*257 || gc != 220*257 || bc != 200*257 {
		t.Fatal("background missing")
	}
	r, _, _, _ = im.At(2, 1).RGBA()
	if r != 0 {
		t.Fatal("frame offset missing")
	}
	_, err = svc.Do(context.Background(), "dither_animate", []byte(`{"input":"partial.gif","output":"wrong.mp4","frames":2,"options":{"width":6}}`))
	if err == nil {
		t.Fatal("wrote GIF to mp4 extension")
	}
	value, err = svc.Do(context.Background(), "dither_animate", []byte(`{"input":"partial.gif","output":"sheet.PNG","frames":2,"options":{"width":6}}`))
	if err != nil {
		t.Fatal(err)
	}
	artifact := value.(Artifact)
	if artifact.Format != "png" {
		t.Fatal(artifact)
	}
	encoded, err := os.ReadFile(filepath.Join(root, "sheet.PNG"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(encoded)); err != nil {
		t.Fatal(err)
	}
	solid := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < len(solid.Pix); i += 4 {
		solid.Pix[i] = 128
		solid.Pix[i+3] = 255
	}
	b.Reset()
	if err := png.Encode(&b, solid); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "solid.png"), b.Bytes(), 0600)
	if _, err := svc.Do(context.Background(), "dither_palette_extract", []byte(`{"input":"solid.png","count":8}`)); err == nil {
		t.Fatal("unusable single-color extraction accepted")
	}
}
func TestProvenanceRetainsIntegerPrecision(t *testing.T) {
	raw := []byte(`{"options":{"seed":9223372036854775806}}`)
	v := provenance(Artifact{}, "dither_render", raw).(Artifact)
	b, err := json.Marshal(v.Parameters)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("9223372036854775806")) {
		t.Fatalf("seed rounded: %s", b)
	}
}
