// Command normalization-showcase creates an original chart and local service previews.
// The comparison separates stored pixel samples from upright, normalized sRGB output.
package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/lennrt/dither-mcp/internal/app"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	paper = color.NRGBA{240, 238, 230, 255}
	ivory = color.NRGBA{250, 249, 245, 255}
	ink   = color.NRGBA{20, 20, 19, 255}
	muted = color.NRGBA{61, 61, 58, 255}
	clay  = color.NRGBA{217, 119, 87, 255}
	line  = color.NRGBA{204, 203, 200, 255}
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func face(data []byte, size float64) font.Face {
	f, err := opentype.Parse(data)
	must(err)
	out, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	must(err)
	return out
}
func label(img draw.Image, f font.Face, x, y int, c color.Color, s string) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: f, Dot: fixed.P(x, y)}
	d.DrawString(s)
}
func fill(img draw.Image, r image.Rectangle, c color.Color) {
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
}
func polygon(img *image.NRGBA, points []image.Point, c color.NRGBA) {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			inside := false
			for i, a := range points {
				b := points[(i+1)%len(points)]
				if (a.Y > y) != (b.Y > y) && float64(x) < float64(b.X-a.X)*float64(y-a.Y)/float64(b.Y-a.Y)+float64(a.X) {
					inside = !inside
				}
			}
			if inside {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}
func chart() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 520, 320))
	fill(img, img.Bounds(), paper)
	f := face(gomono.TTF, 16)
	defer f.Close()
	label(img, f, 28, 35, muted, "UP / RIGHT")
	polygon(img, []image.Point{{160, 62}, {220, 125}, {186, 125}, {186, 220}, {134, 220}, {134, 125}, {100, 125}}, ink)
	polygon(img, []image.Point{{295, 120}, {385, 120}, {385, 88}, {455, 162}, {385, 236}, {385, 202}, {295, 202}}, clay)
	label(img, f, 150, 250, ink, "UP")
	label(img, f, 330, 250, ink, "RIGHT")
	for i, value := range []uint8{64, 128, 192, 240} {
		fill(img, image.Rect(28+i*116, 274, 28+(i+1)*116, 294), color.NRGBA{value, value, value, 255})
	}
	// This translucent marker lets the generator check alpha preservation.
	for y := 19; y < 43; y++ {
		for x := 479; x < 503; x++ {
			if math.Hypot(float64(x-491), float64(y-31)) < 11 {
				img.SetNRGBA(x, y, color.NRGBA{217, 119, 87, 160})
			}
		}
	}
	return img
}
func counterclockwise(src image.Image) image.Image {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.Set(y, b.Dx()-1-x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}
func chunk(kind string, data []byte) []byte {
	b := make([]byte, len(data)+12)
	binary.BigEndian.PutUint32(b, uint32(len(data)))
	copy(b[4:], kind)
	copy(b[8:], data)
	binary.BigEndian.PutUint32(b[len(b)-4:], crc32.ChecksumIEEE(b[4:len(b)-4]))
	return b
}
func encoded(src image.Image, orientation uint16, profile []byte) []byte {
	var b bytes.Buffer
	must(png.Encode(&b, src))
	out := append([]byte(nil), b.Bytes()[:33]...)
	if orientation != 0 {
		exif := make([]byte, 26)
		copy(exif, "II")
		binary.LittleEndian.PutUint16(exif[2:], 42)
		binary.LittleEndian.PutUint32(exif[4:], 8)
		binary.LittleEndian.PutUint16(exif[8:], 1)
		binary.LittleEndian.PutUint16(exif[10:], 274)
		binary.LittleEndian.PutUint16(exif[12:], 3)
		binary.LittleEndian.PutUint32(exif[14:], 1)
		binary.LittleEndian.PutUint16(exif[18:], orientation)
		out = append(out, chunk("eXIf", exif)...)
	}
	if profile != nil {
		var compressed bytes.Buffer
		compressed.WriteString("Original linear RGB\x00\x00")
		z := zlib.NewWriter(&compressed)
		_, err := z.Write(profile)
		must(err)
		must(z.Close())
		out = append(out, chunk("iCCP", compressed.Bytes())...)
	}
	return append(out, b.Bytes()[33:]...)
}
func write(path string, data []byte) {
	must(os.WriteFile(path, data, 0644))
	fmt.Println(path)
}
func rawSamples(data []byte) image.Image {
	// Standard PNG decoding supplies the stored samples for the diagnostic panel.
	// These panels do not represent a color-managed rendering of the source.
	img, err := png.Decode(bytes.NewReader(data))
	must(err)
	return img
}
func thumbnail(dst draw.Image, rect image.Rectangle, src image.Image) {
	fill(dst, rect, ivory)
	bounds := src.Bounds()
	scale := min(float64(rect.Dx()-32)/float64(bounds.Dx()), float64(rect.Dy()-24)/float64(bounds.Dy()))
	w, h := int(float64(bounds.Dx())*scale), int(float64(bounds.Dy())*scale)
	x0, y0 := rect.Min.X+(rect.Dx()-w)/2, rect.Min.Y+(rect.Dy()-h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(x0+x, y0+y, src.At(bounds.Min.X+x*bounds.Dx()/w, bounds.Min.Y+y*bounds.Dy()/h))
		}
	}
}
func call(s *app.Service, name string, input string) any {
	b, err := json.Marshal(map[string]any{"input": input, "width": 520})
	must(err)
	if name == "dither_inspect" {
		b, err = json.Marshal(map[string]any{"input": input})
		must(err)
	}
	result, err := s.Do(context.Background(), name, b)
	must(err)
	return result
}

func main() {
	// Unwind workspace cleanup before reporting a failed generation step.
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintln(os.Stderr, "normalization-showcase:", err)
			os.Exit(1)
		}
	}()
	must(os.MkdirAll("docs/assets/source", 0755))
	profile, err := os.ReadFile("internal/imagemeta/testdata/linear-srgb.icc")
	must(err)
	serif, err := os.ReadFile("docs/assets/fonts/SourceSerif4-Regular.ttf")
	must(err)
	title := face(serif, 48)
	heading := face(serif, 30)
	mono := face(gomono.TTF, 14)
	defer title.Close()
	defer heading.Close()
	defer mono.Close()
	source := chart()
	original := encoded(source, 0, nil)
	rotated := encoded(counterclockwise(source), 6, nil)
	linear := encoded(source, 0, profile)
	write("docs/assets/source/normalization-chart.png", original)
	write("docs/assets/source/normalization-oriented.png", rotated)
	write("docs/assets/source/normalization-linear.png", linear)
	write("docs/assets/source/normalization-linear.icc", profile)
	stage, err := os.MkdirTemp("", "dither-normalization-showcase-")
	must(err)
	defer os.RemoveAll(stage)
	must(os.WriteFile(filepath.Join(stage, "normalization-oriented.png"), rotated, 0600))
	must(os.WriteFile(filepath.Join(stage, "normalization-linear.png"), linear, 0600))
	s, err := app.New(stage, false)
	must(err)
	defer s.Close()
	var previews []image.Image
	var receipts []app.Inspection
	for _, name := range []string{"oriented", "linear"} {
		inspection := call(s, "dither_inspect", "normalization-"+name+".png").(app.Inspection)
		if inspection.Width != 520 || inspection.Height != 320 || inspection.Normalization.WorkingSpace != "srgb" {
			must(fmt.Errorf("unexpected normalization result for %s: %+v", name, inspection))
		}
		if name == "oriented" && (!inspection.Normalization.OrientationApplied || inspection.Normalization.EXIFOrientation != 6) {
			must(fmt.Errorf("the service did not apply EXIF orientation"))
		}
		if name == "linear" && (!inspection.Normalization.ColorConverted || inspection.Normalization.ICCSHA256 == "") {
			must(fmt.Errorf("the service did not apply the selected ICC profile"))
		}
		receipts = append(receipts, inspection)
		preview := call(s, "dither_preview", "normalization-"+name+".png").(app.PreviewResult)
		data, err := base64.StdEncoding.DecodeString(preview.Data)
		must(err)
		write("docs/assets/normalization-"+name+"-preview.png", data)
		previews = append(previews, rawSamples(data))
	}
	neutral := color.NRGBAModel.Convert(previews[1].At(185, 284)).(color.NRGBA)
	if neutral.R < 187 || neutral.R > 189 || neutral.G < 187 || neutral.G > 189 || neutral.B < 187 || neutral.B > 189 {
		must(fmt.Errorf("unexpected linear 128 sample conversion: %+v", neutral))
	}
	alpha := color.NRGBAModel.Convert(previews[1].At(491, 31)).(color.NRGBA).A
	if alpha != 160 {
		must(fmt.Errorf("normalization changed the alpha sample: %d", alpha))
	}
	if color.NRGBAModel.Convert(previews[0].At(160, 150)).(color.NRGBA) != source.NRGBAAt(160, 150) {
		must(fmt.Errorf("upright preview does not match the original chart"))
	}
	proof := struct {
		Tools       []string         `json:"tools"`
		Inspections []app.Inspection `json:"inspections"`
		Linear128   color.NRGBA      `json:"linear_128_srgb"`
		Alpha       uint8            `json:"preserved_alpha"`
	}{[]string{"dither_inspect", "dither_preview"}, receipts, neutral, alpha}
	b, err := json.MarshalIndent(proof, "", "  ")
	must(err)
	write("docs/assets/normalization-inspection.json", append(b, '\n'))
	out := image.NewNRGBA(image.Rect(0, 0, 1200, 1060))
	fill(out, out.Bounds(), paper)
	label(out, title, 48, 80, ink, "Upright images. sRGB colors.")
	label(out, mono, 50, 115, muted, "Actual local service previews from an original Go chart.")
	label(out, mono, 50, 162, muted, "01 / EXIF ORIENTATION")
	label(out, heading, 50, 202, ink, "Stored orientation 6")
	label(out, heading, 634, 202, ink, "Upright preview")
	thumbnail(out, image.Rect(48, 225, 584, 505), rawSamples(rotated))
	thumbnail(out, image.Rect(632, 225, 1168, 505), previews[0])
	label(out, mono, 50, 532, muted, "320 × 520 stored pixels. Rotate clockwise.")
	label(out, mono, 634, 532, muted, "520 × 320 / dither_preview")
	fill(out, image.Rect(48, 569, 1168, 570), line)
	label(out, mono, 50, 614, muted, "02 / INPUT COLOR PROFILE")
	label(out, heading, 50, 654, ink, "Stored linear RGB samples")
	label(out, heading, 634, 654, ink, "Normalized sRGB preview")
	thumbnail(out, image.Rect(48, 677, 584, 957), rawSamples(linear))
	thumbnail(out, image.Rect(632, 677, 1168, 957), previews[1])
	label(out, mono, 50, 984, muted, "Numeric samples shown directly for comparison.")
	label(out, mono, 634, 984, muted, fmt.Sprintf("Linear 128 → sRGB %d. Alpha %d is preserved.", neutral.R, alpha))
	label(out, mono, 50, 1031, muted, "Go service / dither_inspect + dither_preview / original synthetic ICC v4 profile")
	var pngData bytes.Buffer
	must(png.Encode(&pngData, out))
	write("docs/assets/normalization.png", pngData.Bytes())
}
