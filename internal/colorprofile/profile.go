// Package colorprofile converts bounded RGB matrix/TRC profiles to sRGB.
// It implements a media-relative colorimetric transform with channel clipping.
// It does not provide perceptual gamut mapping or black-point compensation.
package colorprofile

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

const (
	// MaxProfileBytes limits each uncompressed ICC profile to 4 MiB.
	MaxProfileBytes = 4 << 20
	// MaxTags bounds the ICC tag table, including descriptive tags.
	MaxTags = 256
	// MaxCurveSamples bounds each sampled tone reproduction curve.
	MaxCurveSamples = 65536
	// MaxImagePixels bounds the image allocation made by Apply.
	MaxImagePixels = 16 << 20
)

// Transform is immutable after construction. Concurrent calls to Apply are safe.
// Use Parse or FromPNG to construct a valid transform.
type Transform struct {
	matrix matrix
	curves [3][]float64
	valid  bool
}

type tag struct {
	name        string
	offset, end int
}

func fail(format string, args ...any) error {
	return fmt.Errorf("colorprofile: "+format, args...)
}

func u32(p []byte) uint32 { return binary.BigEndian.Uint32(p) }
func fixed16(p []byte) float64 {
	return float64(int32(u32(p))) / 65536
}

// Parse reads an ICC v2 or v4 RGB matrix/TRC profile with an XYZ D50 PCS.
// It accepts input, display, and color-space classes. It rejects LUT pipelines.
// Descriptive tags never select a transform or bypass numerical validation.
func Parse(data []byte) (*Transform, error) {
	if len(data) < 132 || len(data) > MaxProfileBytes {
		return nil, fail("ICC profile size must be 132 through %d bytes", MaxProfileBytes)
	}
	if uint64(u32(data)) != uint64(len(data)) || len(data)%4 != 0 {
		return nil, fail("ICC declared size must match the padded profile length")
	}
	if string(data[36:40]) != "acsp" {
		return nil, fail("ICC signature must be acsp")
	}
	if (data[8] != 2 && data[8] != 4) || data[9]>>4 > 9 || data[9]&15 > 9 || data[10] != 0 || data[11] != 0 {
		return nil, fail("only ICC v2 and v4 profiles are supported")
	}
	switch string(data[12:16]) {
	case "scnr", "mntr", "spac":
	default:
		return nil, fail("unsupported ICC profile class %q", data[12:16])
	}
	if string(data[16:20]) != "RGB " || string(data[20:24]) != "XYZ " {
		return nil, fail("ICC input must be RGB with an XYZ PCS")
	}
	if u32(data[64:68]) > 3 {
		return nil, fail("invalid ICC rendering intent")
	}
	for i, v := range d50 {
		if math.Abs(fixed16(data[68+i*4:])-v) > 3.0/65536 {
			return nil, fail("ICC PCS illuminant must be D50")
		}
	}
	if !allZero(data[100:128]) {
		return nil, fail("ICC reserved header bytes must be zero")
	}
	n := uint64(u32(data[128:132]))
	if n == 0 || n > MaxTags || 132+12*n > uint64(len(data)) {
		return nil, fail("ICC tag count must be 1 through %d and fit the profile", MaxTags)
	}
	tableEnd := 132 + int(n)*12
	tags := make(map[string]tag, int(n))
	ranges := make([]tag, 0, int(n))
	for i := range int(n) {
		entry := data[132+i*12 : 144+i*12]
		name := string(entry[:4])
		offset, size := uint64(u32(entry[4:8])), uint64(u32(entry[8:12]))
		if offset%4 != 0 || offset < uint64(tableEnd) || size < 8 || offset+size > uint64(len(data)) {
			return nil, fail("ICC tag %q has an invalid range", name)
		}
		if _, exists := tags[name]; exists {
			return nil, fail("duplicate ICC tag %q", name)
		}
		t := tag{name, int(offset), int(offset + size)}
		raw := data[t.offset:t.end]
		if !allZero(raw[4:8]) {
			return nil, fail("ICC tag %q has nonzero reserved bytes", name)
		}
		if unsupportedTag(name) || unsupportedType(string(raw[:4])) {
			return nil, fail("unsupported ICC LUT, processing, or HDR tag %q", name)
		}
		tags[name] = t
		ranges = append(ranges, t)
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].offset == ranges[j].offset {
			return ranges[i].end < ranges[j].end
		}
		return ranges[i].offset < ranges[j].offset
	})
	cursor := tableEnd
	for i, t := range ranges {
		if i > 0 {
			previous := ranges[i-1]
			if t.offset == previous.offset && t.end == previous.end {
				continue
			}
			if t.offset < previous.end && (t.offset != previous.offset || t.end != previous.end) {
				return nil, fail("ICC tag data overlap without exact sharing")
			}
		}
		if t.offset != cursor {
			return nil, fail("ICC tag data must be contiguous after the tag table")
		}
		paddedEnd := (t.end + 3) &^ 3
		if paddedEnd > len(data) || !allZero(data[t.end:paddedEnd]) {
			return nil, fail("ICC tag %q has invalid padding", t.name)
		}
		cursor = paddedEnd
	}
	if cursor != len(data) {
		return nil, fail("ICC profile has unreferenced trailing data")
	}
	// Shared curve data are legal. The transform retains only immutable numbers.
	var source matrix
	var curves [3][]float64
	for i, prefix := range []string{"r", "g", "b"} {
		t, found := tags[prefix+"XYZ"]
		if !found {
			return nil, fail("ICC matrix requires %sXYZ", prefix)
		}
		xyz, err := parseXYZ(data[t.offset:t.end])
		if err != nil {
			return nil, fail("%sXYZ: %v", prefix, err)
		}
		for row := range 3 {
			source[row][i] = xyz[row]
		}
		t, found = tags[prefix+"TRC"]
		if !found {
			return nil, fail("ICC matrix requires %sTRC", prefix)
		}
		curves[i], err = parseCurve(data[t.offset:t.end])
		if err != nil {
			return nil, fail("%sTRC: %v", prefix, err)
		}
	}
	if _, err := source.inverse(); err != nil {
		return nil, fail("ICC colorant matrix is singular")
	}
	if t, ok := tags["wtpt"]; ok {
		if _, err := parseXYZ(data[t.offset:t.end]); err != nil {
			return nil, fail("wtpt: %v", err)
		}
	}
	if t, ok := tags["chad"]; ok {
		raw := data[t.offset:t.end]
		if len(raw) != 44 || string(raw[:4]) != "sf32" {
			return nil, fail("chad must contain a 3 by 3 sf32 matrix")
		}
		var chad matrix
		for row := range 3 {
			for col := range 3 {
				chad[row][col] = fixed16(raw[8+4*(row*3+col):])
			}
		}
		if _, err := chad.inverse(); err != nil {
			return nil, fail("chad matrix is singular")
		}
		// The source XYZ colorants already include adaptation to the D50 PCS.
		// Applying chad again would adapt the source twice.
	}
	return &Transform{matrix: pcsToSRGB.mul(source), curves: curves, valid: true}, nil
}

func allZero(p []byte) bool {
	for _, v := range p {
		if v != 0 {
			return false
		}
	}
	return true
}

func unsupportedTag(s string) bool {
	return s[:3] == "A2B" || s[:3] == "B2A" || s[:3] == "D2B" || s[:3] == "B2D" || s == "cicp"
}

func unsupportedType(s string) bool {
	switch s {
	case "mft1", "mft2", "mAB ", "mBA ", "mpet", "cicp":
		return true
	}
	return false
}

func parseXYZ(data []byte) ([3]float64, error) {
	if len(data) != 20 || string(data[:4]) != "XYZ " {
		return [3]float64{}, fmt.Errorf("expected one XYZ value")
	}
	return [3]float64{fixed16(data[8:12]), fixed16(data[12:16]), fixed16(data[16:20])}, nil
}
