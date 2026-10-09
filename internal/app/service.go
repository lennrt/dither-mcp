package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/lennrt/dither-mcp/engine"
	"github.com/lennrt/dither-mcp/internal/imagemeta"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type Service struct {
	root       *os.Root
	gate       chan struct{}
	allowVideo bool
}

func New(root string, allowVideo bool) (*Service, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open workspace: %w", err)
	}
	return &Service{root: r, gate: make(chan struct{}, 1), allowVideo: allowVideo}, nil
}
func (s *Service) Close() error { return s.root.Close() }

// Do is the only entry point for transports. Serial admission bounds concurrent
// allocations. Deadlines include queue time. All tools complete their work within the call.
func (s *Service) Do(ctx context.Context, name string, raw []byte) (result any, err error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if len(raw) > 1<<20 {
		return nil, errors.New("arguments exceed 1 MiB")
	}
	defer func() {
		if err == nil {
			result = provenance(result, name, raw)
		}
	}()
	switch name {
	case "dither_catalog":
		var q Empty
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.Catalog(), nil
	case "dither_palettes":
		var q PaletteQuery
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return listPalettes(q)
	case "dither_preview":
		var q PreviewRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.preview(ctx, q)
	case "dither_studio":
		var q StudioRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.studio(ctx, q)
	case "dither_inspect":
		var q InputRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.inspect(ctx, q.Input)
	case "dither_palette_extract":
		var q ExtractRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		im, _, _, err := s.decode(ctx, q.Input)
		if err != nil {
			return nil, err
		}
		if q.Count == 0 {
			q.Count = 8
		}
		p, err := engine.ExtractPalette(ctx, im, q.Count)
		if err != nil {
			return nil, err
		}
		if len(p) < 2 {
			return nil, errors.New("source has fewer than two visible colors. Use a named palette or add a second custom color")
		}
		return map[string]any{"colors": hexColors(p)}, nil
	case "dither_render":
		var q RenderRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.render(ctx, q)
	case "dither_compare":
		var q CompareRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.compare(ctx, q)
	case "dither_batch":
		var q BatchRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.batch(ctx, q)
	case "dither_animate":
		var q AnimateRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.animate(ctx, q)
	case "dither_video":
		var q VideoRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.video(ctx, q)
	case "dither_separate":
		var q RenderRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.separate(ctx, q)
	case "dither_recipe_save":
		var q RecipeSaveRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		if err := validateRecipe(ctx, q.Recipe); err != nil {
			return nil, err
		}
		b, err := json.MarshalIndent(q.Recipe, "", "  ")
		if err != nil {
			return nil, err
		}
		return s.publish(ctx, q.Output, "json", append(b, '\n'), 0, 0, 0, &q.Recipe)
	case "dither_recipe_load":
		var q InputRequest
		if err := StrictJSON(raw, &q); err != nil {
			return nil, err
		}
		return s.loadRecipe(ctx, q.Input)
	default:
		return nil, fmt.Errorf("unknown tool %q", name)
	}
}

func StrictJSON(b []byte, dst any) error {
	if len(bytes.TrimSpace(b)) == 0 || bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return errors.New("arguments must be a JSON object")
	}
	if bytes.TrimSpace(b)[0] != '{' {
		return errors.New("arguments must be a JSON object")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("expected exactly one JSON object")
	}
	return nil
}

func cleanPath(p string) (string, error) {
	if p == "" || len(p) > 4096 || strings.ContainsAny(p, "\x00\\:\r\n") || filepath.IsAbs(p) {
		return "", errors.New("path must be a non-empty relative local path")
	}
	for _, c := range strings.Split(p, "/") {
		if c == ".." {
			return "", errors.New("parent traversal is not allowed")
		}
	}
	p = filepath.Clean(p)
	if p == "." || !filepath.IsLocal(p) {
		return "", errors.New("path must name a file beneath the workspace")
	}
	return p, nil
}
func (s *Service) read(p string, limit int64) ([]byte, error) {
	p, err := cleanPath(p)
	if err != nil {
		return nil, err
	}
	// Refuse special files before opening so a named pipe cannot block admission.
	st, err := s.root.Stat(p)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("input must be a regular file")
	}
	if st.Size() > limit {
		return nil, fmt.Errorf("input exceeds %d byte limit", limit)
	}
	f, err := s.root.OpenFile(p, os.O_RDONLY|readNoBlockFlag, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err = f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("input must be a regular file")
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("input grew beyond byte limit")
	}
	return b, nil
}
func (s *Service) decode(ctx context.Context, p string) (image.Image, []byte, string, error) {
	return s.decodeExpected(ctx, p, "", "source")
}
func decodeBytes(ctx context.Context, b []byte) (image.Image, string, error) {
	im, format, _, err := decodeNormalized(ctx, b)
	return im, format, err
}
func decodeNormalized(ctx context.Context, b []byte) (image.Image, string, imagemeta.Info, error) {
	cfg, f, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, "", imagemeta.Info{}, fmt.Errorf("decode image header: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > engine.MaxDimension || cfg.Height > engine.MaxDimension || int64(cfg.Width)*int64(cfg.Height) > engine.MaxPixels {
		return nil, "", imagemeta.Info{}, errors.New("image dimensions exceed limits")
	}
	metadata, err := imagemeta.Parse(ctx, b, f)
	if err != nil {
		return nil, "", imagemeta.Info{}, err
	}
	im, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, "", imagemeta.Info{}, fmt.Errorf("decode image: %w", err)
	}
	if f == "gif" {
		// Decode returns the first subrectangle. Callers need its logical canvas.
		frame, ok := im.(*image.Paletted)
		if !ok || len(b) < 13 {
			return nil, "", imagemeta.Info{}, errors.New("invalid GIF first frame")
		}
		g := &gif.GIF{Image: []*image.Paletted{frame}, Config: cfg, BackgroundIndex: b[11]}
		canvas := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(gifBackground(g, frame)), image.Point{}, draw.Src)
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		im = canvas
	}
	im, info, err := metadata.Apply(ctx, im)
	return im, f, info, err
}

func (s *Service) inspect(ctx context.Context, p string) (Inspection, error) {
	b, err := s.read(p, MaxBytes)
	if err != nil {
		return Inspection{}, err
	}
	im, f, info, err := decodeNormalized(ctx, b)
	if err != nil {
		return Inspection{}, err
	}
	frames := 1
	if f == "gif" {
		g, err := boundedGIF(b)
		if err != nil {
			return Inspection{}, err
		}
		frames = len(g.Image)
	}
	alpha := false
	bounds := im.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if err := ctx.Err(); err != nil {
			return Inspection{}, err
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := im.At(x, y).RGBA()
			if a != 65535 {
				alpha = true
				break
			}
		}
		if alpha {
			break
		}
	}
	return Inspection{Path: p, Format: f, Width: bounds.Dx(), Height: bounds.Dy(), Frames: frames, Bytes: len(b), SHA256: digest(b), HasAlpha: alpha, Normalization: info}, nil
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func hexColors(p []color.NRGBA) []string {
	r := make([]string, len(p))
	for i, c := range p {
		r[i] = fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return r
}

func validateRecipe(ctx context.Context, r Recipe) error {
	if r.Version != 1 {
		return errors.New("recipe version must be 1")
	}
	p, err := engine.ResolvePalette(r.Palette, r.Colors)
	if err != nil {
		return err
	}
	c := r.Options
	c.Palette = p
	if err := ctx.Err(); err != nil {
		return err
	}
	return engine.ValidateConfig(c)
}
func (s *Service) loadRecipe(ctx context.Context, p string) (Recipe, error) {
	b, err := s.read(p, 1<<20)
	if err != nil {
		return Recipe{}, err
	}
	var r Recipe
	if err = StrictJSON(b, &r); err != nil {
		return r, err
	}
	return r, validateRecipe(ctx, r)
}
func (s *Service) config(ctx context.Context, q RenderRequest) (engine.Config, Recipe, error) {
	c, r, _, err := s.configWithMaskDigest(ctx, q)
	return c, r, err
}

func (s *Service) configWithMaskDigest(ctx context.Context, q RenderRequest) (engine.Config, Recipe, string, error) {
	r := Recipe{Version: 1, Palette: q.Palette, Colors: q.Colors, Options: q.Options}
	if err := validateSourceGuards(q); err != nil {
		return engine.Config{}, r, "", err
	}
	if q.Recipe != "" {
		if q.Palette != "" || len(q.Colors) > 0 || !reflect.DeepEqual(q.Options, engine.Config{}) {
			return engine.Config{}, r, "", errors.New("recipe cannot be combined with inline palette/colors/options")
		}
		var err error
		r, err = s.loadRecipe(ctx, q.Recipe)
		if err != nil {
			return engine.Config{}, r, "", err
		}
	}
	if r.Palette == "" && len(r.Colors) == 0 {
		r.Palette = "mono"
	}
	if r.Options.Algorithm == "" {
		r.Options.Algorithm = "floyd-steinberg"
	}
	p, err := engine.ResolvePalette(r.Palette, r.Colors)
	if err != nil {
		return engine.Config{}, r, "", err
	}
	c := r.Options
	c.Palette = p
	maskSHA256 := ""
	if q.MaskInput != "" {
		im, b, _, err := s.decodeExpected(ctx, q.MaskInput, q.ExpectedMaskSHA256, "mask")
		if err != nil {
			var changed *InputChangedError
			if errors.As(err, &changed) {
				return c, r, "", err
			}
			return c, r, "", fmt.Errorf("mask: %w", err)
		}
		if c.Mask != nil && c.Mask.Shape != "image" {
			return c, r, "", errors.New("mask_input requires options.mask.shape=image or no inline mask")
		}
		maskSHA256 = digest(b)
		c.Mask = &engine.Mask{Shape: "image", Image: im}
		if r.Options.Mask != nil {
			c.Mask.Invert = r.Options.Mask.Invert
		}
	}
	return c, r, maskSHA256, nil
}

func outputFormat(q RenderRequest) (string, error) {
	f := strings.ToLower(q.Format)
	if f == "" {
		f = strings.TrimPrefix(strings.ToLower(filepath.Ext(q.Output)), ".")
	}
	if f == "jpg" {
		f = "jpeg"
	}
	if f == "txt" {
		f = "ascii"
	}
	switch f {
	case "png", "jpeg", "gif", "svg", "pbm", "ascii":
		return f, nil
	default:
		return "", fmt.Errorf("unsupported output format %q", f)
	}
}
func (s *Service) render(ctx context.Context, q RenderRequest) (Artifact, error) {
	if _, err := cleanPath(q.Output); err != nil {
		return Artifact{}, err
	}
	f, err := outputFormat(q)
	if err != nil {
		return Artifact{}, err
	}
	c, r, err := s.config(ctx, q)
	if err != nil {
		return Artifact{}, err
	}
	im, _, _, err := s.decodeExpected(ctx, q.Input, q.ExpectedSourceSHA256, "source")
	if err != nil {
		return Artifact{}, err
	}
	out, err := engine.Process(ctx, im, c)
	if err != nil {
		return Artifact{}, err
	}
	b, err := encodeBounded(f, out, q.DPI)
	if err != nil {
		return Artifact{}, err
	}
	return s.publish(ctx, q.Output, f, b, out.Bounds().Dx(), out.Bounds().Dy(), 1, &r)
}

type boundedBuffer struct {
	buf bytes.Buffer
	max int
}

func (b *boundedBuffer) Len() int      { return b.buf.Len() }
func (b *boundedBuffer) Bytes() []byte { return b.buf.Bytes() }
func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.max-b.Len() {
		return 0, errors.New("encoded output exceeds byte limit")
	}
	return b.buf.Write(p)
}

func (s *Service) publish(ctx context.Context, p, format string, b []byte, w, h, n int, r *Recipe) (Artifact, error) {
	p, err := cleanPath(p)
	if err != nil {
		return Artifact{}, err
	}
	if int64(len(b)) > MaxBytes {
		return Artifact{}, errors.New("output exceeds byte limit")
	}
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	parent := filepath.Dir(p)
	if err := s.root.MkdirAll(parent, 0755); err != nil {
		return Artifact{}, err
	}
	// Open the parent root once: renaming a parent cannot redirect publication.
	dir, err := s.root.OpenRoot(parent)
	if err != nil {
		return Artifact{}, err
	}
	defer dir.Close()
	token := make([]byte, 12)
	if _, err = rand.Read(token); err != nil {
		return Artifact{}, err
	}
	tmp := ".dither-" + hex.EncodeToString(token) + ".tmp"
	f, err := dir.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Artifact{}, err
	}
	defer dir.Remove(tmp)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return Artifact{}, err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return Artifact{}, err
	}
	if err = f.Close(); err != nil {
		return Artifact{}, err
	}
	if err = ctx.Err(); err != nil {
		return Artifact{}, err
	}
	if err = dir.Link(tmp, filepath.Base(p)); err != nil {
		return Artifact{}, fmt.Errorf("publish without overwrite: %w", err)
	}
	return Artifact{Path: filepath.ToSlash(p), Format: format, Width: w, Height: h, Frames: n, Bytes: len(b), SHA256: digest(b), Recipe: r}, nil
}

func (s *Service) batch(ctx context.Context, q BatchRequest) (BatchResult, error) {
	if len(q.Items) < 1 || len(q.Items) > 32 {
		return BatchResult{}, errors.New("batch needs 1..32 items")
	}
	r := BatchResult{Items: make([]BatchItem, 0, len(q.Items))}
	for i, item := range q.Items {
		if err := ctx.Err(); err != nil {
			r.Items = append(r.Items, BatchItem{Index: i, Error: err.Error()})
			break
		}
		a, err := s.render(ctx, item)
		entry := BatchItem{Index: i}
		if err != nil {
			entry.Error = err.Error()
		} else {
			entry.Artifact = &a
		}
		r.Items = append(r.Items, entry)
	}
	return r, nil
}
func (s *Service) compare(ctx context.Context, q CompareRequest) (CompareResult, error) {
	if len(q.Algorithms) < 1 || len(q.Algorithms) > 12 {
		return CompareResult{}, errors.New("comparison needs 1..12 algorithms")
	}
	if q.Columns == 0 {
		q.Columns = 3
	}
	if q.Columns < 1 || q.Columns > 12 {
		return CompareResult{}, errors.New("columns must be 1..12")
	}
	c, r, err := s.config(ctx, q.RenderRequest)
	if err != nil {
		return CompareResult{}, err
	}
	if c.Width == 0 && c.Height == 0 {
		c.Width = 320
	}
	im, _, _, err := s.decodeExpected(ctx, q.Input, q.ExpectedSourceSHA256, "source")
	if err != nil {
		return CompareResult{}, err
	}
	var sheet *image.NRGBA
	result := CompareResult{Cells: make([]Cell, 0, len(q.Algorithms))}
	w, h := 0, 0
	for i, id := range q.Algorithms {
		c.Algorithm = id
		out, err := engine.Process(ctx, im, c)
		if err != nil {
			return result, err
		}
		if sheet == nil {
			w, h = out.Bounds().Dx(), out.Bounds().Dy()
			sw, sh := w*q.Columns, h*((len(q.Algorithms)+q.Columns-1)/q.Columns)
			if int64(sw)*int64(sh) > engine.MaxPixels || sw > engine.MaxDimension || sh > engine.MaxDimension {
				return result, errors.New("comparison sheet exceeds image limits")
			}
			sheet = image.NewNRGBA(image.Rect(0, 0, sw, sh))
		}
		x, y := (i%q.Columns)*w, (i/q.Columns)*h
		draw.Draw(sheet, image.Rect(x, y, x+w, y+h), out, image.Point{}, draw.Src)
		result.Cells = append(result.Cells, Cell{Algorithm: id, X: x, Y: y, Width: w, Height: h})
	}
	f, err := outputFormat(q.RenderRequest)
	if err != nil {
		return result, err
	}
	b, err := encodeBounded(f, sheet, q.DPI)
	if err != nil {
		return result, err
	}
	result.Artifact, err = s.publish(ctx, q.Output, f, b, sheet.Bounds().Dx(), sheet.Bounds().Dy(), 1, &r)
	return result, err
}

// provenance records the complete operation parameters, including motion and
// comparison options that are intentionally outside the reusable still recipe.
func provenance(v any, name string, raw []byte) any {
	enrich := func(a Artifact, n string, b []byte) Artifact {
		a.Operation = n
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		_ = d.Decode(&a.Parameters)
		a.EngineVersion = Version
		return a
	}
	switch a := v.(type) {
	case Artifact:
		return enrich(a, name, raw)
	case CompareResult:
		a.Artifact = enrich(a.Artifact, name, raw)
		return a
	case BatchResult:
		var q BatchRequest
		if json.Unmarshal(raw, &q) == nil {
			for i := range a.Items {
				if a.Items[i].Artifact != nil && a.Items[i].Index < len(q.Items) {
					b, _ := json.Marshal(q.Items[a.Items[i].Index])
					v := enrich(*a.Items[i].Artifact, "dither_render", b)
					a.Items[i].Artifact = &v
				}
			}
		}
		return a
	default:
		return v
	}
}
