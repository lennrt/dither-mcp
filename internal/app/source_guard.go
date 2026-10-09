package app

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"strings"
)

// InputChangedError lets a client invalidate a preview when its source or image
// mask has changed. The service returns it before processing or publication.
type InputChangedError struct {
	Code           string `json:"code"`
	Message        string `json:"message"`
	Path           string `json:"path"`
	ExpectedSHA256 string `json:"expected_sha256"`
	ActualSHA256   string `json:"actual_sha256"`
	cause          error
}

func (e *InputChangedError) Error() string { return e.Code + ": " + e.Message }
func (e *InputChangedError) Unwrap() error { return e.cause }

func validateFingerprint(name, expected string) error {
	if expected == "" {
		return nil
	}
	if len(expected) != 64 {
		return fmt.Errorf("%s must contain exactly 64 hexadecimal digits", name)
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return fmt.Errorf("%s must contain exactly 64 hexadecimal digits", name)
	}
	return nil
}

func validateSourceGuards(q RenderRequest) error {
	if err := validateFingerprint("expected_source_sha256", q.ExpectedSourceSHA256); err != nil {
		return err
	}
	if err := validateFingerprint("expected_mask_sha256", q.ExpectedMaskSHA256); err != nil {
		return err
	}
	if q.ExpectedMaskSHA256 != "" && q.MaskInput == "" {
		return errors.New("expected_mask_sha256 requires mask_input")
	}
	return nil
}

// readExpected reads once. Both the fingerprint and subsequent decoding or
// video staging use this buffer, so a path replacement cannot change the input
// between validation and processing.
func (s *Service) readExpected(p, expected, kind string) ([]byte, error) {
	b, err := s.read(p, MaxBytes)
	if err != nil {
		if expected != "" {
			return nil, &InputChangedError{Code: kind + "_changed", Message: fmt.Sprintf("The %s file could not be read. Refresh the preview before exporting.", kind), Path: p, ExpectedSHA256: strings.ToLower(expected), cause: err}
		}
		return nil, err
	}
	if expected != "" {
		actual := digest(b)
		if !strings.EqualFold(expected, actual) {
			return nil, &InputChangedError{Code: kind + "_changed", Message: fmt.Sprintf("The %s file has changed since the preview. Refresh the preview before exporting.", kind), Path: p, ExpectedSHA256: strings.ToLower(expected), ActualSHA256: actual}
		}
	}
	return b, nil
}

func (s *Service) decodeExpected(ctx context.Context, p, expected, kind string) (image.Image, []byte, string, error) {
	b, err := s.readExpected(p, expected, kind)
	if err != nil {
		return nil, nil, "", err
	}
	im, f, err := decodeBytes(ctx, b)
	return im, b, f, err
}
