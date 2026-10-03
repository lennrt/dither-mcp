package imagemeta

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"

	"github.com/lennrt/dither-mcp/internal/colorprofile"
)

func (m *Metadata) jpeg(ctx context.Context, b []byte) error {
	if len(b) < 2 || b[0] != 0xff || b[1] != 0xd8 {
		return errors.New("invalid JPEG signature")
	}
	var parts [256][]byte
	total, size := 0, 0
	for pos := 2; pos < len(b); {
		if err := ctx.Err(); err != nil {
			return err
		}
		if b[pos] != 0xff {
			return errors.New("invalid JPEG marker")
		}
		for pos < len(b) && b[pos] == 0xff {
			pos++
		}
		if pos == len(b) {
			return errors.New("truncated JPEG marker")
		}
		marker := b[pos]
		pos++
		if marker == 0xd9 {
			break
		}
		if marker == 0x01 || marker >= 0xd0 && marker <= 0xd7 {
			continue
		}
		if pos+2 > len(b) {
			return errors.New("truncated JPEG segment")
		}
		n := int(binary.BigEndian.Uint16(b[pos:]))
		if n < 2 || n > len(b)-pos {
			return errors.New("invalid JPEG segment size")
		}
		payload := b[pos+2 : pos+n]
		pos += n
		switch marker {
		case 0xe1:
			if bytes.HasPrefix(payload, []byte("Exif")) {
				if !bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
					return errors.New("invalid JPEG EXIF identifier")
				}
				if err := m.exif(payload); err != nil {
					return err
				}
			}
		case 0xe2:
			if bytes.HasPrefix(payload, []byte("ICC_PROFILE")) {
				if len(payload) < 15 || payload[11] != 0 {
					return errors.New("invalid JPEG ICC segment")
				}
				seq, count := int(payload[12]), int(payload[13])
				if seq < 1 || seq > count || total != 0 && total != count || parts[seq] != nil {
					return errors.New("invalid or duplicate JPEG ICC sequence")
				}
				total = count
				parts[seq] = payload[14:]
				size += len(parts[seq])
				if size > colorprofile.MaxProfileBytes {
					return errors.New("JPEG ICC profile exceeds 4 MiB")
				}
			}
		case 0xda:
			// Skip entropy bytes, stuffed 0xff bytes, and restart markers. The
			// next marker can introduce another scan or a metadata segment.
			for pos < len(b) {
				if pos&65535 == 0 {
					if err := ctx.Err(); err != nil {
						return err
					}
				}
				if b[pos] != 0xff {
					pos++
					continue
				}
				start := pos
				for pos < len(b) && b[pos] == 0xff {
					pos++
				}
				if pos == len(b) {
					return errors.New("truncated JPEG scan")
				}
				if b[pos] == 0 || b[pos] >= 0xd0 && b[pos] <= 0xd7 {
					pos++
					continue
				}
				pos = start
				break
			}
		}
	}
	if total > 0 {
		profile := make([]byte, 0, size)
		for i := 1; i <= total; i++ {
			if parts[i] == nil {
				return errors.New("missing JPEG ICC segment")
			}
			profile = append(profile, parts[i]...)
		}
		return m.setICC(profile)
	}
	return nil
}

func (m *Metadata) png(ctx context.Context, b []byte) error {
	if len(b) < 8 || string(b[:8]) != "\x89PNG\r\n\x1a\n" {
		return errors.New("invalid PNG signature")
	}
	seen := map[string]bool{}
	seenImage, ended := false, false
	for pos := 8; pos < len(b); {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(b)-pos < 12 {
			return errors.New("truncated PNG chunk")
		}
		n := uint64(binary.BigEndian.Uint32(b[pos:]))
		if n+12 > uint64(len(b)-pos) {
			return errors.New("PNG chunk exceeds input")
		}
		kind := string(b[pos+4 : pos+8])
		payload := b[pos+8 : pos+8+int(n)]
		end := pos + 12 + int(n)
		switch kind {
		case "eXIf", "iCCP", "sRGB", "gAMA", "cHRM", "cICP":
			if seen[kind] {
				return errors.New("duplicate PNG " + kind + " chunk")
			}
			seen[kind] = true
			if kind != "eXIf" && seenImage {
				return errors.New("PNG color metadata must precede image data")
			}
			if crc32.ChecksumIEEE(b[pos+4:end-4]) != binary.BigEndian.Uint32(b[end-4:end]) {
				return errors.New("invalid PNG metadata checksum")
			}
		}
		switch kind {
		case "eXIf":
			if err := m.exif(payload); err != nil {
				return err
			}
		case "iCCP":
			i := bytes.IndexByte(payload, 0)
			if i < 1 || i > 79 || i+2 > len(payload) || payload[i+1] != 0 {
				return errors.New("invalid PNG ICC compression header")
			}
			zr, err := zlib.NewReader(bytes.NewReader(payload[i+2:]))
			if err != nil {
				return errors.New("invalid compressed PNG ICC profile")
			}
			profile, readErr := io.ReadAll(io.LimitReader(zr, colorprofile.MaxProfileBytes+1))
			closeErr := zr.Close()
			if readErr != nil || closeErr != nil {
				return errors.New("invalid compressed PNG ICC profile")
			}
			if err := m.setICC(profile); err != nil {
				return err
			}
		case "sRGB":
			if len(payload) != 1 || payload[0] > 3 {
				return errors.New("invalid PNG sRGB chunk")
			}
			m.srgb = true
		case "gAMA":
			if len(payload) != 4 || binary.BigEndian.Uint32(payload) == 0 {
				return errors.New("invalid PNG gamma")
			}
			m.gamma = float64(binary.BigEndian.Uint32(payload)) / 100000
		case "cHRM":
			if len(payload) != 32 {
				return errors.New("invalid PNG chromaticities")
			}
			var chromas [8]float64
			for i := range chromas {
				chromas[i] = float64(binary.BigEndian.Uint32(payload[i*4:])) / 100000
			}
			m.chromas = &chromas
		case "cICP":
			if !bytes.Equal(payload, []byte{1, 13, 0, 1}) {
				return errors.New("unsupported PNG cICP color space. Convert the source to sRGB")
			}
			m.cicp = true
		case "IDAT", "PLTE":
			seenImage = true
		case "IEND":
			ended = true
		}
		pos = end
		if ended {
			break
		}
	}
	if !ended {
		return errors.New("PNG has no end chunk")
	}
	if m.srgb && len(m.icc) != 0 {
		return errors.New("PNG cannot contain both sRGB and iCCP chunks")
	}
	return nil
}

func (m *Metadata) webp(ctx context.Context, b []byte) error {
	if len(b) < 12 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return errors.New("invalid WebP signature")
	}
	end := uint64(binary.LittleEndian.Uint32(b[4:8])) + 8
	if end < 12 || end > uint64(len(b)) {
		return errors.New("invalid WebP RIFF size")
	}
	for pos := uint64(12); pos < end; {
		if err := ctx.Err(); err != nil {
			return err
		}
		if end-pos < 8 {
			return errors.New("truncated WebP chunk")
		}
		kind := string(b[pos : pos+4])
		n := uint64(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		next := pos + 8 + n + n%2
		if next > end {
			return errors.New("WebP chunk exceeds input")
		}
		payload := b[pos+8 : pos+8+n]
		switch kind {
		case "EXIF":
			if err := m.exif(payload); err != nil {
				return err
			}
		case "ICCP":
			if err := m.setICC(payload); err != nil {
				return err
			}
		case "ANIM", "ANMF":
			return errors.New("animated WebP input is unsupported")
		}
		pos = next
	}
	return nil
}

func (m *Metadata) bmp(b []byte) error {
	if len(b) < 18 || string(b[:2]) != "BM" {
		return errors.New("invalid BMP header")
	}
	n := uint64(binary.LittleEndian.Uint32(b[14:18]))
	if n < 108 {
		return nil
	}
	if n+14 > uint64(len(b)) {
		return errors.New("truncated BMP color header")
	}
	space := binary.LittleEndian.Uint32(b[70:74])
	switch space {
	case 0x73524742, 0x57696e20: // LCS_sRGB and LCS_WINDOWS_COLOR_SPACE.
		m.srgb = true
	case 0x4d424544: // PROFILE_EMBEDDED.
		if n != 124 {
			return errors.New("embedded BMP profiles require a V5 header")
		}
		off := uint64(binary.LittleEndian.Uint32(b[126:130]))
		size := uint64(binary.LittleEndian.Uint32(b[130:134]))
		if off < n || size > colorprofile.MaxProfileBytes || 14+off+size > uint64(len(b)) {
			return errors.New("invalid BMP ICC profile span")
		}
		return m.setICC(b[14+off : 14+off+size])
	case 0x4c494e4b: // PROFILE_LINKED.
		return errors.New("linked BMP color profiles are unsupported. Embed an RGB ICC profile")
	default:
		return errors.New("calibrated or unknown BMP color space is unsupported. Convert the source to sRGB")
	}
	return nil
}

// GIF color-profile application extensions are recognized and rejected. Source
// animation requires one color contract for every composited frame.
func checkGIF(ctx context.Context, b []byte) error {
	if len(b) < 13 {
		return errors.New("truncated GIF header")
	}
	pos := 13
	if b[10]&128 != 0 {
		pos += 3 << (1 + (b[10] & 7))
	}
	for pos < len(b) {
		if err := ctx.Err(); err != nil {
			return err
		}
		kind := b[pos]
		pos++
		switch kind {
		case 0x3b:
			return nil
		case 0x21:
			if pos >= len(b) {
				return errors.New("truncated GIF extension")
			}
			label := b[pos]
			pos++
			if label == 0xff && pos+12 <= len(b) && b[pos] == 11 && string(b[pos+1:pos+12]) == "ICCRGBG1012" {
				return errors.New("GIF ICC profiles are unsupported. Convert the source to sRGB")
			}
		case 0x2c:
			if len(b)-pos < 9 {
				return errors.New("truncated GIF frame")
			}
			packed := b[pos+8]
			pos += 9
			if packed&128 != 0 {
				pos += 3 << (1 + (packed & 7))
			}
			pos++ // LZW minimum code size.
		default:
			return errors.New("invalid GIF block")
		}
		for {
			if pos >= len(b) {
				return errors.New("truncated GIF sub-block")
			}
			n := int(b[pos])
			pos++
			if n == 0 {
				break
			}
			if n > len(b)-pos {
				return errors.New("GIF sub-block exceeds input")
			}
			pos += n
		}
	}
	return errors.New("GIF has no trailer")
}
