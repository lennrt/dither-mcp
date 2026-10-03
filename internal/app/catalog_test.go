package app

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func TestCatalogMediaCapabilities(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		svc, err := New(t.TempDir(), enabled)
		if err != nil {
			t.Fatal(err)
		}
		result, err := svc.Do(context.Background(), "dither_catalog", []byte("{}"))
		svc.Close()
		if err != nil {
			t.Fatal(err)
		}
		catalog, ok := result.(CatalogResult)
		if !ok {
			t.Fatalf("catalog result has type %T", result)
		}
		if catalog.VideoEnabled != enabled || catalog.Media.Video.Enabled != enabled {
			t.Fatal("video capability must reflect startup opt-in")
		}
		if !reflect.DeepEqual(catalog.InputFormats, catalog.Media.Still.InputFormats) || !reflect.DeepEqual(catalog.OutputFormats, catalog.Media.Still.OutputFormats) {
			t.Fatal("legacy format lists must retain their still-image meaning")
		}
		video := catalog.Media.Video
		if !slices.Equal(video.InputContainers, []string{"mp4", "mov", "webm", "matroska", "avi"}) || !slices.Equal(video.OutputFormats, []string{"mp4", "webm", "gif", "png"}) {
			t.Fatalf("video formats: %+v", video)
		}
		if video.Audio || video.DefaultFrames != 48 || video.DefaultFPS != 12 || video.DefaultWidth != 480 || video.MaxFrames != MaxFrames || video.MaxFPS != 50 || video.MaxStartSeconds != 86400 {
			t.Fatalf("video limits/defaults: %+v", video)
		}
		if catalog.Media.Animation.DefaultFrames != 24 || !catalog.Media.Animation.SourceGIFTiming {
			t.Fatal("animation must distinguish generated defaults from source GIF timing")
		}
		print := catalog.Media.Print
		if print.OutputFormat != "zip" || print.PlateFormat != "png" || print.DefaultDPI != 0 || print.Masks || print.PostEffects || print.PressCalibrated {
			t.Fatalf("print capabilities: %+v", print)
		}
		if catalog.Limits.MaxImagePixels != engine.MaxPixels || catalog.Limits.MaxFileBytes != MaxBytes {
			t.Fatal("catalog limits differ from enforced service limits")
		}
		normalization := catalog.Media.Still.Normalization
		if normalization.WorkingSpace != "srgb" || normalization.MaxICCBytes != 4<<20 || normalization.MaxEXIFBytes != 4<<20 || normalization.MaxICCTags != 256 || normalization.MaxCurveSamples != 65536 {
			t.Fatalf("normalization contract: %+v", normalization)
		}
		if !slices.Equal(normalization.OrientationFormats, []string{"jpeg", "png", "webp", "tiff"}) || !slices.Contains(normalization.ICCFormats, "bmp-v5") || len(normalization.PNGPrecedence) != 5 {
			t.Fatal("normalization coverage must be discoverable")
		}
		for _, roles := range [][]FormatRole{catalog.Media.Still.FormatRoles, catalog.Media.Animation.FormatRoles, video.FormatRoles} {
			for _, role := range roles {
				if role.Format == "" || role.MIMEType == "" || role.Role == "" || role.Alpha == "" {
					t.Fatalf("incomplete format role: %+v", role)
				}
			}
		}
		encoded, _ := json.Marshal(catalog)
		var wire map[string]any
		if err := json.Unmarshal(encoded, &wire); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"version", "algorithms", "palette_count", "palette_categories", "palette_discovery", "input_formats", "output_formats", "animation_effects", "video_enabled", "limits", "defaults", "paths"} {
			if _, ok := wire[key]; !ok {
				t.Errorf("legacy catalog key %s disappeared", key)
			}
		}
	}
}
