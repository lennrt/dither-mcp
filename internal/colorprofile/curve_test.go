package colorprofile

import (
	"encoding/binary"
	"math"
	"testing"
)

func curveData(samples ...uint16) []byte {
	data := make([]byte, 12+2*len(samples))
	copy(data, "curv")
	binary.BigEndian.PutUint32(data[8:], uint32(len(samples)))
	for i, value := range samples {
		binary.BigEndian.PutUint16(data[12+i*2:], value)
	}
	return data
}

func parametricData(kind uint16, parameters ...float64) []byte {
	data := make([]byte, 12+4*len(parameters))
	copy(data, "para")
	binary.BigEndian.PutUint16(data[8:], kind)
	for i, value := range parameters {
		binary.BigEndian.PutUint32(data[12+i*4:], uint32(int32(math.Round(value*65536))))
	}
	return data
}

func TestCurveVariants(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		x, y float64
	}{
		{"identity", curveData(), .25, .25},
		{"gamma", curveData(512), .5, .25},
		{"sample-interpolation-left", curveData(0, 16384, 65535), .25, .125},
		{"sample-interpolation-right", curveData(0, 16384, 65535), .75, .625},
		{"type-0", parametricData(0, 2), .25, .0625},
		{"type-1-lower", parametricData(1, 2, 1, -.25), .1, 0},
		{"type-1-upper", parametricData(1, 2, 1, -.25), .5, .0625},
		{"type-2-lower", parametricData(2, 2, 1, -.25, .05), .1, .05},
		{"type-2-upper", parametricData(2, 2, 1, -.25, .05), .5, .1125},
		{"type-3-lower", parametricData(3, 2, 1, 0, .5, .25), .1, .05},
		{"type-3-upper", parametricData(3, 2, 1, 0, .5, .25), .5, .25},
		{"type-4-lower", parametricData(4, 2, 1, 0, .5, .25, .1, .02), .1, .07},
		{"type-4-upper", parametricData(4, 2, 1, 0, .5, .25, .1, .02), .5, .35},
		{"type-4-clipped", parametricData(4, 1, 1, 0, .5, .25, 2, .02), .75, 1},
		{"large-exponent-clipped", parametricData(1, 32767, 32767, 32767), .75, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table, err := parseCurve(tc.data)
			if err != nil {
				t.Fatal(err)
			}
			if value := table[int(math.Round(tc.x*65535))]; !finite(value) || math.Abs(value-tc.y) > 4.0/65535 {
				t.Fatalf("curve value %g, want %g", value, tc.y)
			}
		})
	}
}

func TestCurveBounds(t *testing.T) {
	for _, data := range [][]byte{
		nil, []byte("curv"), curveData(0), parametricData(5),
		parametricData(0, 0), parametricData(1, 2, 0, 0),
		parametricData(3, 2, 1, -.5, .5, .25), parametricData(4, 2, 1, -.5, .5, .25, 0, 0),
	} {
		if _, err := parseCurve(data); err == nil {
			t.Fatalf("invalid curve accepted: %x", data)
		}
	}
	maximum := curveData(make([]uint16, MaxCurveSamples)...)
	if _, err := parseCurve(maximum); err != nil {
		t.Fatal("maximum sampled curve rejected:", err)
	}
	if _, err := parseCurve(curveData(make([]uint16, MaxCurveSamples+1)...)); err == nil {
		t.Fatal("oversized curve accepted")
	}
	data := curveData(100, 200)
	binary.BigEndian.PutUint32(data[8:], math.MaxUint32)
	if _, err := parseCurve(data); err == nil {
		t.Fatal("overflowing sample count accepted")
	}
}
