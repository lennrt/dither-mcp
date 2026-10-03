package engine

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"reflect"
)

type vec3 [3]float64

func (v vec3) add(b vec3) vec3    { return vec3{v[0] + b[0], v[1] + b[1], v[2] + b[2]} }
func (v vec3) sub(b vec3) vec3    { return vec3{v[0] - b[0], v[1] - b[1], v[2] - b[2]} }
func (v vec3) mul(s float64) vec3 { return vec3{v[0] * s, v[1] * s, v[2] * s} }
func (v vec3) dot(b vec3) float64 { return v[0]*b[0] + v[1]*b[1] + v[2]*b[2] }
func clamp(v float64) float64     { return max(0, min(1, v)) }
func byteOf(v float64) uint8      { return uint8(math.Round(clamp(v) * 255)) }
func scalar(v *float64, def float64) float64 {
	if v == nil {
		return def
	}
	return *v
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func checkRange(name string, v, lo, hi float64) error {
	if !finite(v) || v < lo || v > hi {
		return fmt.Errorf("%s must be finite and in [%g,%g]", name, lo, hi)
	}
	return nil
}

func checkImage(img image.Image) error {
	if img == nil {
		return fmt.Errorf("image is nil")
	}
	value := reflect.ValueOf(img)
	if (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) && value.IsNil() {
		return fmt.Errorf("image is nil")
	}
	return checkDimensions(img.Bounds().Dx(), img.Bounds().Dy())
}
func checkDimensions(w, h int) error {
	if w < 1 || h < 1 || w > MaxDimension || h > MaxDimension || w > MaxPixels/h {
		return fmt.Errorf("image dimensions %dx%d exceed limits: each dimension 1–%d, at most %d pixels", w, h, MaxDimension, MaxPixels)
	}
	return nil
}

// Validate checks a recipe without allocation or I/O. Before allocating output,
// Process checks geometry that depends on the input image. These checks cover
// crop intersection and dimensions that preserve the aspect ratio.
func Validate(cfg Config) error { return validate(cfg, true) }

func validate(cfg Config, requireMaskImage bool) error {
	id := cfg.Algorithm
	if id == "" {
		id = "floyd-steinberg"
	}
	if _, ok := algorithmByID(id); !ok {
		return fmt.Errorf("unknown algorithm %q", id)
	}
	if len(cfg.Palette) > 0 {
		if len(cfg.Palette) < 2 || len(cfg.Palette) > 256 {
			return fmt.Errorf("palette must have 2–256 colors")
		}
		seen := map[color.NRGBA]bool{}
		for _, c := range cfg.Palette {
			if c.A != 255 {
				return fmt.Errorf("palette colors must be opaque")
			}
			if seen[c] {
				return fmt.Errorf("duplicate palette color %s", Hex(c))
			}
			seen[c] = true
		}
	}
	if cfg.Width < 0 || cfg.Height < 0 || cfg.Width > MaxDimension || cfg.Height > MaxDimension {
		return fmt.Errorf("width and height must be in [0,%d]", MaxDimension)
	}
	if cfg.Width > 0 && cfg.Height > 0 {
		if err := checkDimensions(cfg.Width, cfg.Height); err != nil {
			return err
		}
	}
	if cfg.PixelScale < 0 || cfg.PixelScale > 256 {
		return fmt.Errorf("pixel_scale must be in [0,256]")
	}
	if cfg.ResizeFilter != "" && cfg.ResizeFilter != "nearest" && cfg.ResizeFilter != "bilinear" {
		return fmt.Errorf("unknown resize_filter %q", cfg.ResizeFilter)
	}
	if cfg.ColorSpace != "" && cfg.ColorSpace != "srgb" && cfg.ColorSpace != "linear-rgb" {
		return fmt.Errorf("unknown color_space %q", cfg.ColorSpace)
	}
	for _, v := range []struct {
		name          string
		value, lo, hi float64
	}{
		{"brightness", cfg.Brightness, -1, 1}, {"contrast", scalar(cfg.Contrast, 1), 0, 4}, {"saturation", scalar(cfg.Saturation, 1), 0, 4}, {"gamma", scalar(cfg.Gamma, 1), 0.1, 8}, {"threshold", scalar(cfg.Threshold, .5), 0, 1}, {"strength", scalar(cfg.Strength, 1), 0, 2},
		{"effects.scanlines", cfg.Effects.Scanlines, 0, 1}, {"effects.crt", cfg.Effects.CRT, 0, 1}, {"effects.noise", cfg.Effects.Noise, 0, 1}, {"effects.pixel_sort_threshold", scalar(cfg.Effects.PixelSortThreshold, .25), 0, 1},
	} {
		if err := checkRange(v.name, v.value, v.lo, v.hi); err != nil {
			return err
		}
	}
	if cfg.Effects.ScanlineSpacing < 0 || cfg.Effects.ScanlineSpacing > 256 {
		return fmt.Errorf("scanline_spacing must be in [0,256]")
	}
	if cfg.Effects.Glitch < 0 || cfg.Effects.Glitch > 1024 {
		return fmt.Errorf("glitch must be in [0,1024]")
	}
	if cfg.Crop != nil {
		r := cfg.Crop
		if r.X < 0 || r.Y < 0 || r.Width < 1 || r.Height < 1 || r.X > MaxDimension || r.Y > MaxDimension || r.Width > MaxDimension || r.Height > MaxDimension {
			return fmt.Errorf("crop must have nonnegative origin and positive bounded dimensions")
		}
	}
	if cfg.Mask != nil {
		m := cfg.Mask
		if m.X < -MaxDimension || m.X > MaxDimension || m.Y < -MaxDimension || m.Y > MaxDimension {
			return fmt.Errorf("mask coordinates exceed dimension limits")
		}
		switch m.Shape {
		case "rectangle":
			if m.Width < 1 || m.Height < 1 || m.Width > MaxDimension || m.Height > MaxDimension {
				return fmt.Errorf("rectangle mask requires positive bounded width and height")
			}
		case "circle":
			if m.Radius < 1 || m.Radius > MaxDimension {
				return fmt.Errorf("circle mask requires radius in [1,%d]", MaxDimension)
			}
		case "image":
			if requireMaskImage || m.Image != nil {
				if err := checkImage(m.Image); err != nil {
					return fmt.Errorf("mask: %w", err)
				}
			}
		default:
			return fmt.Errorf("unknown mask shape %q", m.Shape)
		}
	}
	return nil
}

func process(ctx context.Context, img image.Image, cfg Config) (*image.NRGBA, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := Validate(cfg); err != nil {
		return nil, err
	}
	if err := checkImage(img); err != nil {
		return nil, err
	}
	src := img.Bounds()
	if cfg.Crop != nil {
		r := cfg.Crop
		if r.X >= src.Dx() || r.Y >= src.Dy() || r.Width > src.Dx()-r.X || r.Height > src.Dy()-r.Y {
			return nil, fmt.Errorf("crop must fit wholly inside the source image")
		}
		src = image.Rect(src.Min.X+r.X, src.Min.Y+r.Y, src.Min.X+r.X+r.Width, src.Min.Y+r.Y+r.Height)
	}
	w, h := cfg.Width, cfg.Height
	if w == 0 && h == 0 {
		w, h = src.Dx(), src.Dy()
	} else if w == 0 {
		w = max(1, int(math.Round(float64(src.Dx())*float64(h)/float64(src.Dy()))))
	} else if h == 0 {
		h = max(1, int(math.Round(float64(src.Dy())*float64(w)/float64(src.Dx()))))
	}
	if err := checkDimensions(w, h); err != nil {
		return nil, err
	}
	scale := max(1, cfg.PixelScale)
	gw, gh := (w+scale-1)/scale, (h+scale-1)/scale
	out, err := resample(ctx, img, src, gw, gh, cfg.ResizeFilter == "nearest")
	if err != nil {
		return nil, err
	}
	if err = adjust(ctx, out, cfg); err != nil {
		return nil, err
	}
	palette := cfg.Palette
	if len(palette) == 0 {
		palette, _ = ResolvePalette("mono", nil)
	}
	id := cfg.Algorithm
	if id == "" {
		id = "floyd-steinberg"
	}
	quant := newQuantizer(palette, cfg.ColorSpace == "linear-rgb")
	active := func(x, y int) bool { return out.NRGBAAt(x, y).A > 0 && maskContains(cfg.Mask, x, y, gw, gh) }
	if k, ok := kernels[id]; ok {
		err = diffuse(ctx, out, k, quant, cfg, active)
	} else if id == "riemersma" {
		err = riemersma(ctx, out, quant, cfg, active)
	} else {
		err = ordered(ctx, out, quant, cfg, id, active)
	}
	if err != nil {
		return nil, err
	}
	if err = applyEffects(ctx, out, cfg.Effects, cfg.Seed); err != nil {
		return nil, err
	}
	if gw != w || gh != h {
		out, err = resample(ctx, out, out.Bounds(), w, h, true)
	}
	return out, err
}

// resample uses alpha-correct bilinear interpolation or nearest sampling.
// Coordinates refer to pixel centers. All results have Bounds.Min=(0,0).
func resample(ctx context.Context, src image.Image, r image.Rectangle, w, h int, nearest bool) (*image.NRGBA, error) {
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	sx, sy := float64(r.Dx())/float64(w), float64(r.Dy())/float64(h)
	read := pixelReader(src)
	get := func(x, y int) color.NRGBA {
		return read(r.Min.X+max(0, min(r.Dx()-1, x)), r.Min.Y+max(0, min(r.Dy()-1, y)))
	}
	for y := 0; y < h; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fy := (float64(y)+.5)*sy - .5
		iy := int(math.Floor(fy))
		ty := fy - float64(iy)
		for x := 0; x < w; x++ {
			fx := (float64(x)+.5)*sx - .5
			var c color.NRGBA
			if nearest || (r.Dx() == w && r.Dy() == h) {
				c = get(int(math.Floor(fx+.5)), int(math.Floor(fy+.5)))
			} else {
				ix := int(math.Floor(fx))
				tx := fx - float64(ix)
				cs := [4]color.NRGBA{get(ix, iy), get(ix+1, iy), get(ix, iy+1), get(ix+1, iy+1)}
				weights := [4]float64{(1 - tx) * (1 - ty), tx * (1 - ty), (1 - tx) * ty, tx * ty}
				var premul vec3
				var alpha float64
				for i, p := range cs {
					a := float64(p.A) / 255
					alpha += a * weights[i]
					premul = premul.add(vec3{float64(p.R) / 255, float64(p.G) / 255, float64(p.B) / 255}.mul(a * weights[i]))
				}
				if alpha > 0 {
					c = color.NRGBA{byteOf(premul[0] / alpha), byteOf(premul[1] / alpha), byteOf(premul[2] / alpha), byteOf(alpha)}
				}
			}
			if c.A == 0 {
				c = color.NRGBA{}
			}
			out.SetNRGBA(x, y, c)
		}
	}
	return out, nil
}

func adjust(ctx context.Context, img *image.NRGBA, cfg Config) error {
	contrast, saturation, gamma := scalar(cfg.Contrast, 1), scalar(cfg.Saturation, 1), scalar(cfg.Gamma, 1)
	for y := 0; y < img.Rect.Dy(); y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for x := 0; x < img.Rect.Dx(); x++ {
			c := img.NRGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			v := vec3{float64(c.R) / 255, float64(c.G) / 255, float64(c.B) / 255}
			for k := range v {
				v[k] = clamp((v[k]-.5)*contrast + .5 + cfg.Brightness)
				v[k] = math.Pow(v[k], 1/gamma)
			}
			l := v.dot(vec3{.2126, .7152, .0722})
			s := saturation
			if cfg.Grayscale {
				s = 0
			}
			for k := range v {
				v[k] = clamp(l + (v[k]-l)*s)
				if cfg.Invert {
					v[k] = 1 - v[k]
				}
			}
			img.SetNRGBA(x, y, color.NRGBA{byteOf(v[0]), byteOf(v[1]), byteOf(v[2]), c.A})
		}
	}
	return nil
}
func maskContains(m *Mask, x, y, w, h int) bool {
	if m == nil {
		return true
	}
	inside := false
	switch m.Shape {
	case "rectangle":
		inside = x >= m.X && y >= m.Y && x-m.X < m.Width && y-m.Y < m.Height
	case "circle":
		dx, dy := int64(x-m.X), int64(y-m.Y)
		r := int64(m.Radius)
		inside = dx*dx+dy*dy <= r*r
	case "image":
		r := m.Image.Bounds()
		mx, my := r.Min.X+x*r.Dx()/w, r.Min.Y+y*r.Dy()/h
		c := color.NRGBAModel.Convert(m.Image.At(mx, my)).(color.NRGBA)
		inside = (.2126*float64(c.R)+.7152*float64(c.G)+.0722*float64(c.B))*float64(c.A)/255 >= 127.5
	}
	if m.Invert {
		return !inside
	}
	return inside
}

func linear(v float64) float64 {
	if v <= .04045 {
		return v / 12.92
	}
	return math.Pow((v+.055)/1.055, 2.4)
}
func srgb(v float64) float64 {
	if v <= .0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1/2.4) - .055
}

type quantizer struct {
	colors  []color.NRGBA
	vectors []vec3
	linear  bool
}

func newQuantizer(colors []color.NRGBA, isLinear bool) quantizer {
	q := quantizer{colors: colors, vectors: make([]vec3, len(colors)), linear: isLinear}
	for i, c := range colors {
		q.vectors[i] = q.vector(c)
	}
	return q
}
func (q quantizer) vector(c color.NRGBA) vec3 {
	v := vec3{float64(c.R) / 255, float64(c.G) / 255, float64(c.B) / 255}
	if q.linear {
		for i := range v {
			v[i] = linear(v[i])
		}
	}
	return v
}
func (q quantizer) nearest(v vec3) int {
	best := 0
	distance := math.Inf(1)
	for i, p := range q.vectors {
		delta := v.sub(p)
		d := delta.dot(delta)
		if d < distance {
			best = i
			distance = d
		}
	}
	return best
}

// pair selects the two nearest palette entries. It projects the source onto
// their color-space segment. A threshold determines whether to choose the second
// entry. This method keeps arbitrary palettes within their defined RGB colors.
func (q quantizer) pair(v vec3, t, strength float64) int {
	a, b := 0, -1
	da, db := math.Inf(1), math.Inf(1)
	for i, p := range q.vectors {
		d := v.sub(p)
		ds := d.dot(d)
		if ds < da {
			b = a
			db = da
			a = i
			da = ds
		} else if ds < db {
			b = i
			db = ds
		}
	}
	if da < 1e-18 || b < 0 || strength == 0 {
		return a
	}
	delta := q.vectors[b].sub(q.vectors[a])
	amount := clamp(v.sub(q.vectors[a]).dot(delta) / delta.dot(delta))
	// strength=1 uses the original threshold pattern. Zero chooses the nearest color.
	threshold := .5 + (t-.5)*strength
	if amount > threshold {
		return b
	}
	return a
}

// ValidateConfig is the named recipe-validation entry point for service adapters.
func ValidateConfig(cfg Config) error { return validate(cfg, false) }
