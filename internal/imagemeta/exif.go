package imagemeta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/lennrt/dither-mcp/internal/colorprofile"
)

func (m *Metadata) exif(b []byte) error {
	if m.hasEXIF {
		return errors.New("duplicate EXIF metadata")
	}
	m.hasEXIF = true
	if len(b) > MaxEXIFBytes {
		return errors.New("EXIF metadata exceeds 4 MiB")
	}
	// JPEG includes the identifier. WebP encoders use both representations.
	b = bytes.TrimPrefix(b, []byte("Exif\x00\x00"))
	return m.tiff(b, false)
}

// Only IFD0 describes the selected still image. Thumbnail and later-page
// directories are never followed, so their orientation cannot replace IFD0.
func (m *Metadata) tiff(b []byte, readICC bool) error {
	if len(b) < 8 {
		return errors.New("truncated TIFF metadata header")
	}
	var order binary.ByteOrder
	switch string(b[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return errors.New("invalid TIFF byte order")
	}
	if order.Uint16(b[2:4]) != 42 {
		return errors.New("unsupported TIFF metadata version")
	}
	off := uint64(order.Uint32(b[4:8]))
	if off < 8 || off+2 > uint64(len(b)) {
		return errors.New("invalid first TIFF directory offset")
	}
	n := uint64(order.Uint16(b[off : off+2]))
	if n > MaxIFDEntries || off+2+12*n+4 > uint64(len(b)) {
		return errors.New("invalid or oversized TIFF metadata directory")
	}
	seenOrientation, seenICC := false, false
	for i := uint64(0); i < n; i++ {
		e := b[off+2+12*i : off+2+12*(i+1)]
		tag, kind, count := order.Uint16(e), order.Uint16(e[2:]), uint64(order.Uint32(e[4:]))
		switch tag {
		case 0x0112:
			if seenOrientation || kind != 3 || count != 1 {
				return errors.New("EXIF orientation must be one SHORT value")
			}
			seenOrientation = true
			m.orientation = int(order.Uint16(e[8:10]))
			if m.orientation < 1 || m.orientation > 8 {
				return errors.New("EXIF orientation must be from 1 through 8")
			}
		case 34675:
			if !readICC {
				continue
			}
			if seenICC || kind != 7 || count == 0 || count > colorprofile.MaxProfileBytes {
				return errors.New("invalid or oversized TIFF ICC profile")
			}
			seenICC = true
			if count <= 4 {
				return m.setICC(e[8 : 8+count])
			}
			start := uint64(order.Uint32(e[8:]))
			if start < 8 || start+count > uint64(len(b)) {
				return errors.New("TIFF ICC profile offset exceeds input")
			}
			if err := m.setICC(b[start : start+count]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Metadata) setICC(b []byte) error {
	if len(m.icc) != 0 {
		return errors.New("duplicate ICC profile")
	}
	if len(b) == 0 || len(b) > colorprofile.MaxProfileBytes {
		return fmt.Errorf("ICC profile must contain 1 through %d bytes", colorprofile.MaxProfileBytes)
	}
	m.icc = b
	return nil
}
