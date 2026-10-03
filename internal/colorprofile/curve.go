package colorprofile

import (
	"encoding/binary"
	"fmt"
	"math"
)

func parseCurve(data []byte) ([]float64, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("tone curve is truncated")
	}
	var evaluate func(float64) float64
	switch string(data[:4]) {
	case "curv":
		n := uint64(u32(data[8:12]))
		if n > MaxCurveSamples || 12+2*n != uint64(len(data)) {
			return nil, fmt.Errorf("curve sample count or length is invalid")
		}
		switch n {
		case 0:
			evaluate = func(x float64) float64 { return x }
		case 1:
			gamma := float64(binary.BigEndian.Uint16(data[12:14])) / 256
			if gamma <= 0 {
				return nil, fmt.Errorf("gamma must be positive")
			}
			evaluate = func(x float64) float64 { return math.Pow(x, gamma) }
		default:
			evaluate = func(x float64) float64 {
				p := x * float64(n-1)
				i := min(uint64(p), n-2)
				a := float64(binary.BigEndian.Uint16(data[12+2*i:])) / 65535
				b := float64(binary.BigEndian.Uint16(data[14+2*i:])) / 65535
				return a + (b-a)*(p-float64(i))
			}
		}
	case "para":
		kind := int(binary.BigEndian.Uint16(data[8:10]))
		counts := [5]int{1, 3, 4, 5, 7}
		if kind >= len(counts) || len(data) != 12+4*counts[kind] || data[10] != 0 || data[11] != 0 {
			return nil, fmt.Errorf("parametric curve type or length is invalid")
		}
		var p [7]float64
		for i := range counts[kind] {
			p[i] = fixed16(data[12+4*i:])
		}
		if p[0] <= 0 || (kind > 0 && p[1] <= 0) {
			return nil, fmt.Errorf("parametric gamma and scale must be positive")
		}
		if kind >= 3 && p[4] <= 1 && p[1]*max(0, p[4])+p[2] < -1e-12 {
			return nil, fmt.Errorf("parametric power branch has a negative base")
		}
		evaluate = func(x float64) float64 {
			switch kind {
			case 0:
				return math.Pow(x, p[0])
			case 1, 2:
				offset := 0.0
				if kind == 2 {
					offset = p[3]
				}
				if x < -p[2]/p[1] {
					return offset
				}
				return math.Pow(max(0, p[1]*x+p[2]), p[0]) + offset
			case 3:
				if x < p[4] {
					return p[3] * x
				}
				return math.Pow(max(0, p[1]*x+p[2]), p[0])
			default:
				// ICC.1:2022 Table 68 corrects the type-4 table in ICC.1:2010.
				if x < p[4] {
					return p[3]*x + p[6]
				}
				return math.Pow(max(0, p[1]*x+p[2]), p[0]) + p[5]
			}
		}
	default:
		return nil, fmt.Errorf("only curv and para tone curves are supported")
	}
	return curveTable(evaluate), nil
}

func clamp(x float64) float64 { return min(1, max(0, x)) }

func curveTable(evaluate func(float64) float64) []float64 {
	values := make([]float64, 65536)
	for i := range values {
		// ICC parametric output values outside [0,1] must be clipped.
		values[i] = clamp(evaluate(float64(i) / 65535))
	}
	return values
}

func srgbDecode(x float64) float64 {
	if x <= .04045 {
		return x / 12.92
	}
	return math.Pow((x+.055)/1.055, 2.4)
}

func srgbEncode(x float64) float64 {
	x = clamp(x)
	if x <= .0031308 {
		return 12.92 * x
	}
	return 1.055*math.Pow(x, 1.0/2.4) - .055
}
