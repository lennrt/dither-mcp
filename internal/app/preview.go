package app

import (
	"context"
	"encoding/base64"
	"errors"
	"image"
)

type PreviewRequest struct {
	Input string `json:"input" jsonschema:"Read this image relative to the configured workspace. GIF previews use the first frame on the logical canvas."`
	Width int    `json:"width,omitempty" jsonschema:"Use a preview width from 1 through 1024 pixels. Omission or zero selects 512. Preview dimensions do not exceed 1024 pixels."`
}
type PreviewResult struct {
	Path     string `json:"path"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	MIMEType string `json:"mime_type"`
	Data     string `json:"data"`
}

func (s *Service) preview(ctx context.Context, q PreviewRequest) (PreviewResult, error) {
	if q.Width == 0 {
		q.Width = 512
	}
	if q.Width < 1 || q.Width > 1024 {
		return PreviewResult{}, errors.New("preview width must be 1..1024")
	}
	im, _, _, err := s.decode(ctx, q.Input)
	if err != nil {
		return PreviewResult{}, err
	}
	bounds := im.Bounds()
	w := min(q.Width, bounds.Dx())
	h := max(1, bounds.Dy()*w/bounds.Dx())
	if h > 1024 {
		w = max(1, w*1024/h)
		h = 1024
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		if err := ctx.Err(); err != nil {
			return PreviewResult{}, err
		}
		for x := 0; x < w; x++ {
			out.Set(x, y, im.At(bounds.Min.X+x*bounds.Dx()/w, bounds.Min.Y+y*bounds.Dy()/h))
		}
	}
	b, err := encodeBounded("png", out, 0)
	if err != nil {
		return PreviewResult{}, err
	}
	if len(b) > 2<<20 {
		return PreviewResult{}, errors.New("preview exceeds 2 MiB. Request a smaller width")
	}
	return PreviewResult{Path: q.Input, Width: w, Height: h, MIMEType: "image/png", Data: base64.StdEncoding.EncodeToString(b)}, nil
}
