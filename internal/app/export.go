package app

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"math"
	"strings"

	"github.com/lennrt/dither-mcp/engine"
)

func encodeBounded(format string, im image.Image, dpi int) ([]byte, error) {
	if dpi != 0 && (dpi < 36 || dpi > 2400) {
		return nil, errors.New("dpi must be 0 (unspecified) or 36..2400")
	}
	if dpi != 0 && format != "png" {
		return nil, errors.New("dpi metadata is supported only for PNG")
	}
	b := &boundedBuffer{max: int(MaxBytes) - 21}
	if err := engine.Encode(format, b, im); err != nil {
		return nil, err
	}
	data := b.Bytes()
	if dpi != 0 {
		data = pngDPI(data, dpi)
	}
	return data, nil
}
func pngDPI(b []byte, dpi int) []byte {
	if len(b) < 33 {
		return b
	}
	chunk := make([]byte, 21)
	binary.BigEndian.PutUint32(chunk, 9)
	copy(chunk[4:8], "pHYs")
	ppm := uint32(math.Round(float64(dpi) / 0.0254))
	binary.BigEndian.PutUint32(chunk[8:12], ppm)
	binary.BigEndian.PutUint32(chunk[12:16], ppm)
	chunk[16] = 1
	binary.BigEndian.PutUint32(chunk[17:], crc32.ChecksumIEEE(chunk[4:17]))
	r := make([]byte, 0, len(b)+21)
	r = append(r, b[:33]...)
	r = append(r, chunk...)
	return append(r, b[33:]...)
}

func (s *Service) separate(ctx context.Context, q RenderRequest) (Artifact, error) {
	if q.Format != "" && q.Format != "zip" {
		return Artifact{}, errors.New("separations output is a ZIP archive")
	}
	if q.DPI != 0 && (q.DPI < 36 || q.DPI > 2400) {
		return Artifact{}, errors.New("dpi must be 36..2400")
	}
	c, r, err := s.config(ctx, q)
	if err != nil {
		return Artifact{}, err
	}
	if len(c.Palette) > 16 {
		return Artifact{}, errors.New("separations support at most 16 inks")
	}
	if c.Mask != nil || c.Effects != (engine.Effects{}) {
		return Artifact{}, errors.New("separations require no mask or post-effects to preserve exact palette membership")
	}
	im, _, _, err := s.decode(ctx, q.Input)
	if err != nil {
		return Artifact{}, err
	}
	out, err := engine.Process(ctx, im, c)
	if err != nil {
		return Artifact{}, err
	}
	b := &boundedBuffer{max: int(MaxBytes)}
	z := zip.NewWriter(b)
	inks := make([]map[string]any, 0, len(c.Palette))
	bounds := out.Bounds()
	for n, ink := range c.Palette {
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		mask := image.NewNRGBA(bounds)
		count := 0
		for y := 0; y < bounds.Dy(); y++ {
			if err := ctx.Err(); err != nil {
				return Artifact{}, err
			}
			for x := 0; x < bounds.Dx(); x++ {
				p := out.NRGBAAt(x, y)
				v := uint8(255)
				if p.A != 0 && p.R == ink.R && p.G == ink.G && p.B == ink.B {
					v = 0
					count++
				}
				mask.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
			}
		}
		file := fmt.Sprintf("ink-%02d-%02x%02x%02x.png", n+1, ink.R, ink.G, ink.B)
		data, err := encodeBounded("png", mask, q.DPI)
		if err != nil {
			return Artifact{}, err
		}
		w, err := z.Create(file)
		if err != nil {
			return Artifact{}, err
		}
		if _, err = w.Write(data); err != nil {
			return Artifact{}, err
		}
		inks = append(inks, map[string]any{"file": file, "color": hexColors([]color.NRGBA{ink})[0], "pixels": count})
	}
	manifest, err := json.MarshalIndent(map[string]any{"version": 1, "kind": "spot-color-separations", "width": bounds.Dx(), "height": bounds.Dy(), "dpi": q.DPI, "recipe": r, "inks": inks, "note": "Black pixels mark ink coverage. White pixels mark unprinted areas. The plates provide no halftone-angle or press calibration."}, "", "  ")
	if err != nil {
		return Artifact{}, err
	}
	w, err := z.Create("manifest.json")
	if err != nil {
		return Artifact{}, err
	}
	if _, err = w.Write(manifest); err != nil {
		return Artifact{}, err
	}
	if err = z.Close(); err != nil {
		return Artifact{}, err
	}
	return s.publish(ctx, q.Output, "zip", b.Bytes(), bounds.Dx(), bounds.Dy(), 1, &r)
}

// boundedGIF checks the container before DecodeAll allocates frame images.
// Every full logical screen counts against the aggregate frame budget, even
// when the encoded frame is a tiny delta rectangle.
func boundedGIF(b []byte) (*gif.GIF, error) {
	if len(b) < 13 || (!bytes.Equal(b[:6], []byte("GIF87a")) && !bytes.Equal(b[:6], []byte("GIF89a"))) {
		return nil, errors.New("invalid GIF header")
	}
	width, height := int(binary.LittleEndian.Uint16(b[6:8])), int(binary.LittleEndian.Uint16(b[8:10]))
	pixels := int64(width) * int64(height)
	if width == 0 || height == 0 || pixels > engine.MaxPixels || width > engine.MaxDimension || height > engine.MaxDimension {
		return nil, errors.New("GIF dimensions exceed limits")
	}
	pos := 13
	if b[10]&128 != 0 {
		pos += 3 * (1 << ((b[10] & 7) + 1))
	}
	frames := 0
	skip := func() error {
		for {
			if pos >= len(b) {
				return errors.New("truncated GIF blocks")
			}
			n := int(b[pos])
			pos++
			if n == 0 {
				return nil
			}
			if pos+n > len(b) {
				return errors.New("truncated GIF block")
			}
			pos += n
		}
	}
	for pos < len(b) {
		tag := b[pos]
		pos++
		switch tag {
		case 0x3b:
			if frames == 0 {
				return nil, errors.New("GIF contains no frames")
			}
			return gif.DecodeAll(bytes.NewReader(b))
		case 0x21:
			if pos >= len(b) {
				return nil, errors.New("truncated GIF extension")
			}
			pos++
			if err := skip(); err != nil {
				return nil, err
			}
		case 0x2c:
			if pos+9 > len(b) {
				return nil, errors.New("truncated GIF frame")
			}
			left, top := int(binary.LittleEndian.Uint16(b[pos:])), int(binary.LittleEndian.Uint16(b[pos+2:]))
			w, h := int(binary.LittleEndian.Uint16(b[pos+4:])), int(binary.LittleEndian.Uint16(b[pos+6:]))
			if w == 0 || h == 0 || left+w > width || top+h > height {
				return nil, errors.New("GIF frame exceeds logical screen")
			}
			packed := b[pos+8]
			pos += 9
			if packed&128 != 0 {
				pos += 3 * (1 << ((packed & 7) + 1))
			}
			if pos >= len(b) {
				return nil, errors.New("truncated GIF palette")
			}
			pos++
			frames++
			if frames > MaxFrames || pixels*int64(frames) > MaxFramePixels {
				return nil, errors.New("GIF frame budget exceeded")
			}
			if err := skip(); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("invalid GIF block")
		}
	}
	return nil, errors.New("GIF trailer missing")
}

// gifBackground resolves the GIF logical-screen background. If a frame
// uses the background index for its transparent entry, clearing restores transparency.
// Otherwise, clearing restores the opaque global background. Without a global
// table, the surrounding canvas remains transparent. This convention preserves
// web GIFs and the background of opaque partial-frame GIFs.
func gifBackground(g *gif.GIF, frame *image.Paletted) color.Color {
	index := int(g.BackgroundIndex)
	if frame != nil && index < len(frame.Palette) {
		_, _, _, a := frame.Palette[index].RGBA()
		if a == 0 {
			return color.NRGBA{}
		}
	}
	if palette, ok := g.Config.ColorModel.(color.Palette); ok && index < len(palette) {
		return palette[index]
	}
	return color.NRGBA{}
}

func compositeGIF(ctx context.Context, g *gif.GIF) ([]image.Image, error) {
	canvas := image.NewNRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))
	if len(g.Image) > 0 {
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(gifBackground(g, g.Image[0])), image.Point{}, draw.Src)
	}
	out := make([]image.Image, 0, len(g.Image))
	for i, frame := range g.Image {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		disposal := byte(0)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		var before *image.NRGBA
		if disposal == gif.DisposalPrevious {
			before = image.NewNRGBA(canvas.Bounds())
			copy(before.Pix, canvas.Pix)
		}
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		snapshot := image.NewNRGBA(canvas.Bounds())
		copy(snapshot.Pix, canvas.Pix)
		out = append(out, snapshot)
		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, frame.Bounds(), image.NewUniform(gifBackground(g, frame)), image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			canvas = before
		}
	}
	return out, nil
}

// frameDimensions mirrors the engine's output sizing without allocating pixels.
// It checks source-dependent crops and aspect ratios before an animation
// accumulates buffers. The engine independently checks each frame again.
func frameDimensions(bounds image.Rectangle, c engine.Config, frames int) (int, int, error) {
	if err := engine.ValidateConfig(c); err != nil {
		return 0, 0, err
	}
	if frames < 1 || frames > MaxFrames {
		return 0, 0, errors.New("frame count must be 1..120")
	}
	sw, sh := bounds.Dx(), bounds.Dy()
	if sw < 1 || sh < 1 {
		return 0, 0, errors.New("source image is empty")
	}
	if c.Crop != nil {
		r := c.Crop
		if r.X >= sw || r.Y >= sh || r.Width > sw-r.X || r.Height > sh-r.Y {
			return 0, 0, errors.New("crop must fit wholly inside the source image")
		}
		sw, sh = r.Width, r.Height
	}
	w, h := c.Width, c.Height
	if w == 0 && h == 0 {
		w, h = sw, sh
	} else if w == 0 {
		w = max(1, int(math.Round(float64(sw)*float64(h)/float64(sh))))
	} else if h == 0 {
		h = max(1, int(math.Round(float64(sh)*float64(w)/float64(sw))))
	}
	if w < 1 || h < 1 || w > engine.MaxDimension || h > engine.MaxDimension || int64(w)*int64(h) > engine.MaxPixels {
		return 0, 0, errors.New("output image dimensions exceed limits")
	}
	if int64(w)*int64(h)*int64(frames) > MaxFramePixels {
		return 0, 0, errors.New("animation exceeds total frame-pixel budget")
	}
	return w, h, nil
}

func validateSpriteDimensions(w, h, frames, columns int) error {
	if columns == 0 {
		columns = 6
	}
	if columns < 1 || columns > 16 {
		return errors.New("sprite columns must be 1..16")
	}
	sw, sh := int64(w)*int64(columns), int64(h)*int64((frames+columns-1)/columns)
	if sw > engine.MaxDimension || sh > engine.MaxDimension || sw*sh > engine.MaxPixels {
		return errors.New("spritesheet exceeds image bounds")
	}
	return nil
}

func (s *Service) animate(ctx context.Context, q AnimateRequest) (Artifact, error) {
	c, r, err := s.config(ctx, q.RenderRequest)
	if err != nil {
		return Artifact{}, err
	}
	im, b, f, err := s.decode(ctx, q.Input)
	if err != nil {
		return Artifact{}, err
	}
	if q.Effect == "" {
		if f == "gif" {
			q.Effect = "source"
		} else {
			q.Effect = "wave"
		}
	}
	if q.Frames == 0 {
		q.Frames = 24
	}
	if q.FPS == 0 {
		q.FPS = 12
	}
	if q.FPS < 1 || q.FPS > 50 || q.Frames < 1 || q.Frames > MaxFrames {
		return Artifact{}, errors.New("fps must be 1..50 and frames 1..120")
	}
	if math.IsNaN(q.Amplitude) || math.IsInf(q.Amplitude, 0) || q.Amplitude < 0 || q.Amplitude > 1 {
		return Artifact{}, errors.New("amplitude must be in [0,1]")
	}
	if q.Amplitude == 0 {
		q.Amplitude = .15
	}
	switch q.Effect {
	case "source", "wave", "orbit", "pulse", "noise", "palette-cycle":
	default:
		return Artifact{}, errors.New("unknown animation effect")
	}
	if _, err := cleanPath(q.Output); err != nil {
		return Artifact{}, err
	}
	if q.DPI != 0 {
		return Artifact{}, errors.New("animation does not support dpi metadata. Export print stills with dither_render")
	}
	format := strings.ToLower(q.Format)
	if format == "" {
		format = strings.ToLower(stringsExt(q.Output))
	}
	if format != "gif" && format != "png" {
		return Artifact{}, errors.New("animation format must be gif or png (sprite sheet)")
	}
	sources := []image.Image{im}
	delays := []int(nil)
	loop := 0
	var sourceGIF *gif.GIF
	sourceBounds := im.Bounds()
	if q.Effect == "source" && f == "gif" {
		g, err := boundedGIF(b)
		if err != nil {
			return Artifact{}, err
		}
		sourceGIF = g
		sourceBounds = image.Rect(0, 0, g.Config.Width, g.Config.Height)
		q.Frames = len(g.Image)
		delays = g.Delay
		loop = g.LoopCount
	}
	if c.Width == 0 && c.Height == 0 {
		c.Width = 480
	}
	w, h, err := frameDimensions(sourceBounds, c, q.Frames)
	if err != nil {
		return Artifact{}, err
	}
	if format == "png" {
		if err = validateSpriteDimensions(w, h, q.Frames, q.Columns); err != nil {
			return Artifact{}, err
		}
	}
	if sourceGIF != nil {
		sources, err = compositeGIF(ctx, sourceGIF)
		if err != nil {
			return Artifact{}, err
		}
	}
	frames := make([]*image.NRGBA, 0, q.Frames)
	var pixels int64
	for i := 0; i < q.Frames; i++ {
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		cfg := c
		phase := 2 * math.Pi * float64(i) / float64(q.Frames)
		source := sources[i%len(sources)]
		switch q.Effect {
		case "wave", "orbit":
			source = warp(ctx, source, phase, q.Amplitude, q.Effect)
		case "pulse":
			cfg.Brightness = math.Max(-1, math.Min(1, c.Brightness+q.Amplitude*math.Sin(phase)))
		case "noise":
			cfg.Seed = c.Seed + int64(i)
			cfg.Algorithm = "random"
		case "palette-cycle":
			cfg.Palette = append([]color.NRGBA(nil), c.Palette...)
			shift := i * len(c.Palette) / q.Frames
			for j := range cfg.Palette {
				cfg.Palette[j] = c.Palette[(j+shift)%len(c.Palette)]
			}
		}
		// Palette-cycle recolors the quantized indices below. Permuting the palette
		// search order would not provide the intended recoloring effect.
		if q.Effect == "palette-cycle" {
			cfg.Palette = c.Palette
		}
		frame, err := engine.Process(ctx, source, cfg)
		if err != nil {
			return Artifact{}, err
		}
		pixels += int64(frame.Bounds().Dx()) * int64(frame.Bounds().Dy())
		if pixels > MaxFramePixels {
			return Artifact{}, errors.New("animation exceeds total frame-pixel budget")
		}
		if q.Effect == "palette-cycle" {
			shift := i * len(c.Palette) / q.Frames
			for y := 0; y < frame.Bounds().Dy(); y++ {
				if err := ctx.Err(); err != nil {
					return Artifact{}, err
				}
				for x := 0; x < frame.Bounds().Dx(); x++ {
					p := frame.NRGBAAt(x, y)
					for j, ink := range c.Palette {
						if p.R == ink.R && p.G == ink.G && p.B == ink.B {
							v := c.Palette[(j+shift)%len(c.Palette)]
							v.A = p.A
							frame.SetNRGBA(x, y, v)
							break
						}
					}
				}
			}
		}
		frames = append(frames, frame)
	}
	return s.publishFrames(ctx, q.Output, format, frames, delays, q.FPS, loop, q.Columns, &r)
}
func stringsExt(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '.' {
			return p[i+1:]
		}
		if p[i] == '/' {
			break
		}
	}
	return ""
}
func warp(ctx context.Context, im image.Image, phase, amplitude float64, kind string) image.Image {
	b := im.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	w, h := b.Dx(), b.Dy()
	for y := 0; y < h; y++ {
		if ctx.Err() != nil {
			return dst
		}
		for x := 0; x < w; x++ {
			dx, dy := 0, 0
			if kind == "wave" {
				dx = int(math.Sin(float64(y)/float64(h)*4*math.Pi+phase) * amplitude * float64(w) / 4)
			} else {
				dx = int(math.Cos(phase) * amplitude * float64(w) / 4)
				dy = int(math.Sin(phase) * amplitude * float64(h) / 4)
			}
			sx, sy := (x+dx+w)%w, (y+dy+h)%h
			dst.Set(x, y, im.At(b.Min.X+sx, b.Min.Y+sy))
		}
	}
	return dst
}

func (s *Service) publishFrames(ctx context.Context, path, format string, frames []*image.NRGBA, delays []int, fps, loop, columns int, r *Recipe) (Artifact, error) {
	w, h := frames[0].Bounds().Dx(), frames[0].Bounds().Dy()
	if format == "png" {
		if columns == 0 {
			columns = 6
		}
		if columns < 1 || columns > 16 {
			return Artifact{}, errors.New("sprite columns must be 1..16")
		}
		sw, sh := w*columns, h*((len(frames)+columns-1)/columns)
		if int64(sw)*int64(sh) > engine.MaxPixels || sw > engine.MaxDimension || sh > engine.MaxDimension {
			return Artifact{}, errors.New("spritesheet exceeds image bounds")
		}
		sheet := image.NewNRGBA(image.Rect(0, 0, sw, sh))
		for i, frame := range frames {
			x, y := i%columns*w, i/columns*h
			draw.Draw(sheet, image.Rect(x, y, x+w, y+h), frame, image.Point{}, draw.Src)
		}
		b, err := encodeBounded("png", sheet, 0)
		if err != nil {
			return Artifact{}, err
		}
		return s.publish(ctx, path, "png", b, sw, sh, len(frames), r)
	}
	if format != "gif" {
		return Artifact{}, errors.New("animation format must be gif or png (sprite sheet)")
	}
	g := &gif.GIF{LoopCount: loop}
	for i, frame := range frames {
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		encoded, err := encodeBounded("gif", frame, 0)
		if err != nil {
			return Artifact{}, err
		}
		decoded, err := gif.Decode(bytes.NewReader(encoded))
		if err != nil {
			return Artifact{}, err
		}
		p, ok := decoded.(*image.Paletted)
		if !ok {
			return Artifact{}, errors.New("GIF encoder returned non-indexed image")
		}
		g.Image = append(g.Image, p)
		// GIF stores centiseconds. Distribute rounding error across frames so
		// non-divisor rates such as 12 fps keep the requested overall duration.
		delay := int(math.Round(100*float64(i+1)/float64(fps))) - int(math.Round(100*float64(i)/float64(fps)))
		if len(delays) > i {
			delay = delays[i]
		}
		g.Delay = append(g.Delay, delay)
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
	}
	b := &boundedBuffer{max: int(MaxBytes)}
	if err := gif.EncodeAll(b, g); err != nil {
		return Artifact{}, err
	}
	return s.publish(ctx, path, "gif", b.Bytes(), w, h, len(frames), r)
}
