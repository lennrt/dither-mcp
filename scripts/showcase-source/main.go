// Command showcase-source creates original, deterministic artwork for the demos.
// It uses only the Go standard library. The repository license covers these images.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type vec struct{ x, y, z float64 }

func (a vec) add(b vec) vec       { return vec{a.x + b.x, a.y + b.y, a.z + b.z} }
func (a vec) sub(b vec) vec       { return vec{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a vec) mul(s float64) vec   { return vec{a.x * s, a.y * s, a.z * s} }
func (a vec) dot(b vec) float64   { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a vec) cross(b vec) vec     { return vec{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x} }
func (a vec) norm() vec           { return a.mul(1 / math.Sqrt(a.dot(a))) }
func mix(a, b vec, t float64) vec { return a.mul(1 - t).add(b.mul(t)) }
func clamp(x float64) float64     { return math.Max(0, math.Min(1, x)) }
func rgb(r, g, b float64) vec     { return vec{r / 255, g / 255, b / 255} }
func noise(x, y, z float64) float64 {
	return .42*math.Sin(x*11.2+math.Sin(z*4.2)+y*9.7) + .23*math.Sin(x*29.4+y*17.1+z*23.3) + .12*math.Sin(x*83.7-y*68.2+z*61.9) + .06*math.Sin(x*187.3+y*161.7-z*153.3)
}

type object struct {
	c, s, u, v, w vec
	kind          int
	base          vec
}
type hit struct {
	t           float64
	p, n, local vec
	obj         *object
}
type scene struct {
	objects            []object
	eye, target, light vec
	theme              int
}

func ellipsoid(c, s, axis vec, kind int, base vec) object {
	v := axis.norm()
	reference := vec{0, 0, 1}
	if math.Abs(v.dot(reference)) > .9 {
		reference = vec{1, 0, 0}
	}
	u := v.cross(reference).norm()
	w := u.cross(v).norm()
	return object{c, s, u, v, w, kind, base}
}
func (o *object) intersect(origin, direction vec) (hit, bool) {
	p := origin.sub(o.c)
	q := vec{p.dot(o.u) / o.s.x, p.dot(o.v) / o.s.y, p.dot(o.w) / o.s.z}
	d := vec{direction.dot(o.u) / o.s.x, direction.dot(o.v) / o.s.y, direction.dot(o.w) / o.s.z}
	a := d.dot(d)
	b := q.dot(d)
	c := q.dot(q) - 1
	discriminant := b*b - a*c
	if discriminant < 0 {
		return hit{}, false
	}
	t := (-b - math.Sqrt(discriminant)) / a
	if t < .001 {
		t = (-b + math.Sqrt(discriminant)) / a
	}
	if t < .001 {
		return hit{}, false
	}
	l := q.add(d.mul(t))
	n := o.u.mul(l.x / o.s.x).add(o.v.mul(l.y / o.s.y)).add(o.w.mul(l.z / o.s.z)).norm()
	return hit{t, origin.add(direction.mul(t)), n, l, o}, true
}
func (s *scene) intersect(origin, direction vec) (hit, bool) {
	best := hit{t: 1e20}
	found := false
	for i := range s.objects {
		if h, ok := s.objects[i].intersect(origin, direction); ok && h.t < best.t {
			best = h
			found = true
		}
	}
	if direction.y < -.00001 {
		t := -origin.y / direction.y
		if t > .001 && t < best.t {
			best = hit{t: t, p: origin.add(direction.mul(t)), n: vec{0, 1, 0}}
			found = true
		}
	}
	return best, found
}
func (s *scene) shade(origin, direction vec) vec {
	h, ok := s.intersect(origin, direction)
	if !ok {
		t := clamp(.5 + direction.y*.75)
		if s.theme == 1 {
			return mix(rgb(55, 74, 83), rgb(180, 159, 133), 1-t)
		}
		return mix(rgb(192, 143, 107), rgb(59, 83, 83), t)
	}
	base := rgb(184, 174, 151)
	specular := .05
	rough := noise(h.p.x*1.5, h.p.y*1.5, h.p.z*1.5)
	if h.obj != nil {
		base = h.obj.base
		switch h.obj.kind {
		case 1: // Waxy foliage has ribs and a mottled surface.
			vein := math.Exp(-math.Abs(h.local.x) * 90)
			side := math.Pow(math.Max(0, math.Cos(h.local.y*39-math.Abs(h.local.x)*24)), 14)
			base = base.mul(.85 + rough*.12 + vein*.35 + side*.08)
			specular = .28
		case 2: // Vertical flutes shape the unglazed ceramic.
			angle := math.Atan2(h.local.x, h.local.z)
			flute := math.Sin(angle * 40)
			base = base.mul(.92 + .06*flute + rough*.025)
			specular = .09
		case 3: // Stone has mineral bands.
			strata := math.Sin(h.p.y*40 + noise(h.p.x, h.p.y*.1, h.p.z)*3)
			base = base.mul(.86 + strata*.09 + rough*.1)
			specular = .035
		case 4:
			specular = .42
			base = base.mul(.92 + rough*.03)
		case 5:
			base = base.mul(.9 + rough*.07)
		}
	} else {
		if s.theme == 1 {
			bands := math.Sin(h.p.z*1.5 + noise(h.p.x*.4, 0, h.p.z*.4)*3)
			base = mix(rgb(180, 140, 105), rgb(200, 172, 137), .5+.3*bands)
			base = base.mul(1 + .03*noise(h.p.x*4, 0, h.p.z*4))
		} else {
			base = rgb(163, 151, 125).mul(.96 + rough*.04)
		}
	}
	light := s.light.sub(h.p).norm()
	diffuse := math.Max(0, h.n.dot(light))
	shadow := 1.0
	for i := 0; i < 3; i++ {
		offset := vec{float64(i-1) * .35, 0, float64((i*2)%3-1) * .25}
		l := s.light.add(offset).sub(h.p)
		if sh, yes := s.intersect(h.p.add(h.n.mul(.003)), l.norm()); yes && sh.t < math.Sqrt(l.dot(l)) {
			shadow -= .245
		}
	}
	ambient := .22 + .08*math.Max(0, h.n.y)
	result := base.mul(ambient + diffuse*.9*shadow)
	halfway := light.sub(direction).norm()
	shine := math.Pow(math.Max(0, h.n.dot(halfway)), 48) * specular * shadow
	result = result.add(rgb(255, 230, 192).mul(shine))
	// Atmospheric perspective blends distant ground with the sky.
	if h.obj == nil {
		mist := 1 - math.Exp(-h.t*.013)
		result = mix(result, rgb(169, 150, 126), mist)
	}
	return result
}
func garden() scene {
	s := scene{eye: vec{3.6, 2.75, 5.3}, target: vec{.05, 1.65, 0}, light: vec{-3, 7, 4}}
	add := func(o object) { s.objects = append(s.objects, o) }
	add(ellipsoid(vec{-.2, .58, 0}, vec{.78, .65, .78}, vec{0, 1, 0}, 2, rgb(205, 159, 116)))
	add(ellipsoid(vec{-.2, 1.15, 0}, vec{.69, .07, .69}, vec{0, 1, 0}, 3, rgb(43, 47, 37)))
	// Multiple stems lean away from one another. Each leaf has its own angle and scale.
	for branch := 0; branch < 5; branch++ {
		angle := float64(branch) * 2.399
		height := 1.35 + float64(branch%3)*.32
		base := vec{-.2, 1.1, 0}
		tip := vec{-.2 + math.Cos(angle)*.57, 1.1 + height, math.Sin(angle) * .55}
		add(ellipsoid(mix(base, tip, .5), vec{.025, height * .52, .025}, tip.sub(base), 5, rgb(91, 103, 66)))
		for j := 0; j < 7; j++ {
			t := .25 + float64(j)*.10
			root := mix(base, tip, t)
			phi := angle + float64(j)*2.4
			axis := vec{math.Cos(phi) * .6, .23 + float64(j)*.035, math.Sin(phi) * .6}
			size := .37 - .012*float64(j)
			add(ellipsoid(root.add(axis.mul(.48)), vec{size * .47, size * 1.28, .052}, axis, 1, mix(rgb(74, 142, 114), rgb(134, 164, 104), float64(branch)/6)))
		}
	}
	add(ellipsoid(vec{1.25, .36, .4}, vec{.58, .37, .43}, vec{.2, 1, .1}, 3, rgb(124, 130, 120)))
	add(ellipsoid(vec{-1.7, .23, .4}, vec{.45, .22, .35}, vec{0, 1, 0}, 3, rgb(191, 171, 137)))
	add(ellipsoid(vec{1.35, 2.7, -2.8}, vec{1, 1, 1}, vec{0, 1, 0}, 4, rgb(227, 175, 102)))
	return s
}
func desert() scene {
	s := scene{eye: vec{4.2, 2.8, 7.7}, target: vec{0, 1.25, 0}, light: vec{-4, 7, 2}, theme: 1}
	for i := 0; i < 24; i++ {
		a := float64(i) * 2.399
		r := .7 + float64((i*7)%17)*.12
		height := .13 + float64((i*11)%13)*.035
		p := vec{math.Cos(a) * r, height * .78, math.Sin(a)*r - .7}
		size := .18 + float64((i*3)%9)*.048
		s.objects = append(s.objects, ellipsoid(p, vec{size, height, size * .8}, vec{.2, 1, .1}, 3, mix(rgb(106, 85, 73), rgb(185, 122, 87), float64(i%6)/6)))
	}
	for i := 0; i < 6; i++ {
		y := .33 + float64(i)*.41
		width := .56 - float64(i)*.045
		p := vec{-.2 + math.Sin(float64(i)*.9)*.10, y, -.3}
		s.objects = append(s.objects, ellipsoid(p, vec{width, .33, width * .72}, vec{.15, 1, .05}, 3, mix(rgb(193, 115, 80), rgb(141, 103, 83), float64(i)/6)))
	}
	s.objects = append(s.objects, ellipsoid(vec{1.4, 3.0, -3.5}, vec{1.08, 1.08, 1.08}, vec{0, 1, 0}, 4, rgb(216, 183, 129)))
	return s
}
func render(s scene, path string, w, h int) error {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	forward := s.target.sub(s.eye).norm()
	right := forward.cross(vec{0, 1, 0}).norm()
	up := right.cross(forward).norm()
	aspect := float64(w) / float64(h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sum := vec{}
			for k := 0; k < 4; k++ {
				sx := (float64(x)+.25+float64(k%2)*.5)/float64(w)*2 - 1
				sy := 1 - (float64(y)+.25+float64(k/2)*.5)/float64(h)*2
				ray := forward.add(right.mul(sx * aspect * .44)).add(up.mul(sy * .44)).norm()
				sum = sum.add(s.shade(s.eye, ray).mul(.25))
			}
			vignette := 1 - .12*(math.Pow((float64(x)/float64(w)-.5)*2, 2)+math.Pow((float64(y)/float64(h)-.5)*2, 2))
			sum = sum.mul(vignette)
			grain := noise(float64(x)*.14, float64(y)*.15, 0) * .003
			channel := func(v float64) uint8 { return uint8(clamp(math.Pow(math.Max(0, v), 1/1.15)+grain)*255 + .5) }
			img.SetNRGBA(x, y, color.NRGBA{channel(sum.x), channel(sum.y), channel(sum.z), 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
func main() {
	out := flag.String("out", "docs/assets/source", "Output directory")
	width := flag.Int("width", 960, "Image width")
	flag.Parse()
	if *width < 64 || *width > 4096 {
		panic("width must be between 64 and 4096")
	}
	if err := os.MkdirAll(*out, 0755); err != nil {
		panic(err)
	}
	for _, item := range []struct {
		name  string
		scene scene
	}{{"moon-garden", garden()}, {"mineral-nocturne", desert()}} {
		path := filepath.Join(*out, item.name+".png")
		if err := render(item.scene, path, *width, *width*2/3); err != nil {
			panic(err)
		}
		fmt.Println(path)
	}
}
