package imagemeta

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"strings"
	"testing"
)

func reviewBMP(headerSize, space uint32) []byte {
	b := make([]byte, 138)
	copy(b, "BM")
	binary.LittleEndian.PutUint32(b[14:], headerSize)
	binary.LittleEndian.PutUint32(b[70:], space)
	return b
}

func TestReviewBMPProfileAdmission(t *testing.T) {
	for name, space := range map[string]uint32{"srgb": 0x73524742, "windows-srgb": 0x57696e20} {
		t.Run(name, func(t *testing.T) {
			for _, size := range []uint32{108, 124} {
				m, err := Parse(context.Background(), reviewBMP(size, space), "bmp")
				if err != nil || m.source != "bmp-srgb" || m.transform != nil {
					t.Fatalf("header %d: metadata=%+v error=%v", size, m, err)
				}
			}
		})
	}
	linked := reviewBMP(124, 0x4c494e4b)
	binary.LittleEndian.PutUint32(linked[126:], 124)
	path := []byte("/outside-workspace/private-profile.icc\x00")
	binary.LittleEndian.PutUint32(linked[130:], uint32(len(path)))
	linked = append(linked, path...)
	if _, err := Parse(context.Background(), linked, "bmp"); err == nil || !strings.Contains(err.Error(), "linked BMP") {
		t.Fatalf("linked profile must fail without resolving its path: %v", err)
	}
	for name, b := range map[string][]byte{
		"calibrated":      reviewBMP(124, 0),
		"unknown-space":   reviewBMP(124, 0xffffffff),
		"huge-header":     reviewBMP(0xfffffff0, 0x73524742),
		"v4-embedded":     reviewBMP(108, 0x4d424544),
		"truncated-v5":    reviewBMP(124, 0x4d424544)[:133],
		"missing-profile": reviewBMP(124, 0x4d424544),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(context.Background(), b, "bmp"); err == nil {
				t.Fatal("accepted invalid BMP color metadata")
			}
		})
	}
	for _, off := range []uint32{0, 123, 0xfffffff0} {
		b := reviewBMP(124, 0x4d424544)
		binary.LittleEndian.PutUint32(b[126:], off)
		binary.LittleEndian.PutUint32(b[130:], 132)
		b = append(b, make([]byte, 132)...)
		if _, err := Parse(context.Background(), b, "bmp"); err == nil {
			t.Fatalf("accepted profile offset %d", off)
		}
	}
}

func TestReviewEXIFDirectoryAdmission(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		full := exifBytes(6, order)
		for length := 0; length < len(full); length++ {
			if _, err := Parse(context.Background(), full[:length], "tiff"); err == nil {
				t.Fatalf("accepted truncated %T TIFF directory at %d bytes", order, length)
			}
		}
		oversized := append([]byte(nil), full...)
		order.PutUint16(oversized[8:], MaxIFDEntries+1)
		oversized = append(oversized, make([]byte, 12*MaxIFDEntries)...)
		if _, err := Parse(context.Background(), oversized, "tiff"); err == nil {
			t.Fatal("accepted a directory above the entry limit")
		}
		// A following image directory belongs to a thumbnail or another page.
		// Its orientation must not replace the selected image's orientation.
		next := exifBytes(3, order)[8:]
		withThumbnail := append([]byte(nil), full...)
		order.PutUint32(withThumbnail[22:], uint32(len(withThumbnail)))
		withThumbnail = append(withThumbnail, next...)
		m, err := Parse(context.Background(), withThumbnail, "tiff")
		if err != nil || m.orientation != 6 {
			t.Fatalf("thumbnail replaced IFD0 for %T: %+v, %v", order, m, err)
		}
		for _, format := range []string{"jpeg", "png", "webp"} {
			var b []byte
			switch format {
			case "jpeg":
				b = jpegContainer(jpegSegment(0xe1, append([]byte("Exif\x00\x00"), withThumbnail...)))
			case "png":
				b = pngContainer(pngChunk("eXIf", withThumbnail))
			case "webp":
				b = webpContainer(riffChunk("EXIF", withThumbnail))
			}
			m, err := Parse(context.Background(), b, format)
			if err != nil || m.orientation != 6 {
				t.Fatalf("%s %T disagrees with IFD0: %+v, %v", format, order, m, err)
			}
		}
	}
}

func TestReviewCICPPrecedesProfileInternals(t *testing.T) {
	unsupported := append([]byte(nil), profileFixture(t)...)
	copy(unsupported[16:20], "CMYK")
	cicp := pngChunk("cICP", []byte{1, 13, 0, 1})
	src := image.NewNRGBA64(image.Rect(0, 0, 1, 1))
	want := color.NRGBA64{R: 12001, G: 22002, B: 32003, A: 23005}
	src.SetNRGBA64(0, 0, want)
	for _, profile := range [][]byte{unsupported, []byte("not an ICC profile")} {
		if _, err := Parse(context.Background(), pngContainer(iccp(profile)), "png"); err == nil {
			t.Fatal("control fixture must fail when its profile determines color")
		}
		for _, chunks := range [][][]byte{{cicp, iccp(profile)}, {iccp(profile), cicp}} {
			m, err := Parse(context.Background(), pngContainer(chunks...), "png")
			if err != nil {
				t.Fatal(err)
			}
			out, info, err := m.Apply(context.Background(), src)
			if err != nil || info.ColorSource != "png-cicp-srgb" || info.ColorConverted || info.ICCSHA256 != "" {
				t.Fatalf("ignored ICC reported as selected: %+v, %v", info, err)
			}
			if got := out.At(0, 0).(color.NRGBA64); got != want {
				t.Fatalf("cICP should retain all sRGB samples: got %+v want %+v", got, want)
			}
		}
	}
	for _, chunks := range [][][]byte{
		{cicp, pngChunk("iCCP", []byte("Profile\x00\x00invalid zlib"))},
		{cicp, iccp(unsupported), pngChunk("sRGB", []byte{0})},
	} {
		if _, err := Parse(context.Background(), pngContainer(chunks...), "png"); err == nil {
			t.Fatal("cICP must not conceal invalid container metadata or conflicting declarations")
		}
	}
}

func TestReviewJPEGMetadataBetweenScans(t *testing.T) {
	profile := profileFixture(t)
	part := func(seq byte, data []byte) []byte {
		return jpegSegment(0xe2, append([]byte{'I', 'C', 'C', '_', 'P', 'R', 'O', 'F', 'I', 'L', 'E', 0, seq, 2}, data...))
	}
	// These are metadata-scanner fixtures, not complete JPEG decoder fixtures.
	// Stuffed marker bytes and restart markers must remain within entropy data.
	scan1 := append(jpegSegment(0xda, []byte{1, 1, 0, 0, 0, 0}),
		[]byte{1, 2, 0xff, 0, 0xe1, 3, 0xff, 0xd0, 4, 0xff, 0xff, 0xd1, 5}...)
	scan2 := append(jpegSegment(0xda, []byte{1, 1, 0, 1, 63, 0}), []byte{6, 0xff, 0, 7}...)
	exif := jpegSegment(0xe1, append([]byte("Exif\x00\x00"), exifBytes(8, binary.BigEndian)...))
	b := jpegContainer(part(2, profile[len(profile)/2:]), scan1, exif, scan2, part(1, profile[:len(profile)/2]))
	m, err := Parse(context.Background(), b, "jpeg")
	if err != nil || m.orientation != 8 || m.source != "embedded-icc" || !bytes.Equal(m.icc, profile) {
		t.Fatalf("metadata after entropy was lost: %+v, %v", m, err)
	}
	if _, err := Parse(context.Background(), jpegContainer(exif, scan1, exif), "jpeg"); err == nil {
		t.Fatal("accepted duplicate EXIF separated by entropy data")
	}
	truncated := append([]byte{0xff, 0xd8}, scan1...)
	truncated = append(truncated, 0xff)
	if _, err := Parse(context.Background(), truncated, "jpeg"); err == nil {
		t.Fatal("accepted a truncated marker at the end of a scan")
	}
}
