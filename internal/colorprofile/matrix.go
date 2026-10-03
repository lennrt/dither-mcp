package colorprofile

import (
	"fmt"
	"math"
)

type matrix [3][3]float64

var d50 = [3]float64{.9642, 1, .8249}
var srgbChromaticities = [8]float64{.3127, .3290, .64, .33, .30, .60, .15, .06}

// The linear Bradford cone matrix appears in ICC.1 Annex E.
var bradford = matrix{{.8951, .2664, -.1614}, {-.7502, 1.7135, .0367}, {.0389, -.0685, 1.0296}}

var pcsToSRGB = func() matrix {
	m, err := primaryMatrix(srgbChromaticities)
	if err != nil {
		panic(err)
	}
	white, _ := xyXYZ(srgbChromaticities[0], srgbChromaticities[1])
	a, err := adaptation(white, d50)
	if err != nil {
		panic(err)
	}
	inverse, err := a.mul(m).inverse()
	if err != nil {
		panic(err)
	}
	return inverse
}()

func (m matrix) vector(v [3]float64) (out [3]float64) {
	for row := range 3 {
		for col := range 3 {
			out[row] += m[row][col] * v[col]
		}
	}
	return out
}

func (m matrix) mul(n matrix) (out matrix) {
	for row := range 3 {
		for col := range 3 {
			for k := range 3 {
				out[row][col] += m[row][k] * n[k][col]
			}
		}
	}
	return out
}

func (m matrix) inverse() (matrix, error) {
	scale := 0.0
	for _, row := range m {
		for _, value := range row {
			if !finite(value) {
				return matrix{}, fmt.Errorf("matrix must be finite")
			}
			scale = max(scale, math.Abs(value))
		}
	}
	if scale == 0 {
		return matrix{}, fmt.Errorf("matrix is singular")
	}
	for row := range 3 {
		for col := range 3 {
			m[row][col] /= scale
		}
	}
	a, b, c := m[0][0], m[0][1], m[0][2]
	d, e, f := m[1][0], m[1][1], m[1][2]
	g, h, i := m[2][0], m[2][1], m[2][2]
	det := a*(e*i-f*h) - b*(d*i-f*g) + c*(d*h-e*g)
	if math.IsNaN(det) || math.IsInf(det, 0) || math.Abs(det) < 1e-10 {
		return matrix{}, fmt.Errorf("matrix is singular or numerically unstable")
	}
	out := matrix{{e*i - f*h, c*h - b*i, b*f - c*e}, {f*g - d*i, a*i - c*g, c*d - a*f}, {d*h - e*g, b*g - a*h, a*e - b*d}}
	for row := range 3 {
		for col := range 3 {
			out[row][col] = out[row][col] / det / scale
			if !finite(out[row][col]) {
				return matrix{}, fmt.Errorf("matrix inverse must be finite")
			}
		}
	}
	return out, nil
}

func xyXYZ(x, y float64) ([3]float64, error) {
	if !finite(x) || !finite(y) || x < 0 || y <= 0 || x+y > 1+1e-12 {
		return [3]float64{}, fmt.Errorf("chromaticities must be finite, nonnegative, and satisfy y > 0 and x+y <= 1")
	}
	return [3]float64{x / y, 1, max(0, 1-x-y) / y}, nil
}

func primaryMatrix(chroma [8]float64) (matrix, error) {
	white, err := xyXYZ(chroma[0], chroma[1])
	if err != nil {
		return matrix{}, err
	}
	var m matrix
	for col := range 3 {
		xyz, err := xyXYZ(chroma[2+2*col], chroma[3+2*col])
		if err != nil {
			return matrix{}, err
		}
		for row := range 3 {
			m[row][col] = xyz[row]
		}
	}
	inverse, err := m.inverse()
	if err != nil {
		return matrix{}, err
	}
	scale := inverse.vector(white)
	for col := range 3 {
		if !finite(scale[col]) || scale[col] <= 0 {
			return matrix{}, fmt.Errorf("white point must lie inside the primary triangle")
		}
		for row := range 3 {
			m[row][col] *= scale[col]
			if !finite(m[row][col]) {
				return matrix{}, fmt.Errorf("primary matrix must be finite")
			}
		}
	}
	if _, err := m.inverse(); err != nil {
		return matrix{}, err
	}
	return m, nil
}

func adaptation(source, destination [3]float64) (matrix, error) {
	src, dst := bradford.vector(source), bradford.vector(destination)
	var diagonal matrix
	for i := range 3 {
		if !finite(src[i]) || src[i] <= 1e-12 || !finite(dst[i]) || dst[i] <= 0 {
			return matrix{}, fmt.Errorf("white point has invalid Bradford cone responses")
		}
		diagonal[i][i] = dst[i] / src[i]
	}
	inverse, err := bradford.inverse()
	if err != nil {
		return matrix{}, err
	}
	return inverse.mul(diagonal).mul(bradford), nil
}

func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// FromPNG creates a transform from PNG gAMA and cHRM values.
// A zero gamma uses the sRGB transfer function. Other values use exponent 1/gamma.
// Nil chromaticities select sRGB primaries and D65. The array order is
// white x/y, red x/y, green x/y, and blue x/y.
func FromPNG(gamma float64, chromaticities *[8]float64) (*Transform, error) {
	if !finite(gamma) || gamma < 0 || (gamma > 0 && !finite(1/gamma)) {
		return nil, fail("PNG gamma must be zero or a finite positive value with a finite reciprocal")
	}
	chroma := srgbChromaticities
	if chromaticities != nil {
		chroma = *chromaticities
	}
	m, err := primaryMatrix(chroma)
	if err != nil {
		return nil, fail("invalid PNG chromaticities: %v", err)
	}
	white, err := xyXYZ(chroma[0], chroma[1])
	if err != nil {
		return nil, fail("invalid PNG white point: %v", err)
	}
	a, err := adaptation(white, d50)
	if err != nil {
		return nil, fail("invalid PNG adaptation: %v", err)
	}
	evaluate := srgbDecode
	if gamma > 0 {
		evaluate = func(x float64) float64 { return math.Pow(x, 1/gamma) }
	}
	table := curveTable(evaluate)
	return &Transform{matrix: pcsToSRGB.mul(a).mul(m), curves: [3][]float64{table, table, table}, valid: true}, nil
}
