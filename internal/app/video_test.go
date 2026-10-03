package app

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestVideoDisabledAndInvalid(t *testing.T) {
	svc, err := New(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	_, err = svc.Do(context.Background(), "dither_video", []byte(`{"input":"no.mp4","output":"out.mp4"}`))
	if err == nil {
		t.Fatal("disabled video accepted")
	}
}
func TestVideoIntegration(t *testing.T) {
	if os.Getenv("DITHER_TEST_VIDEO") != "1" {
		t.Skip("set DITHER_TEST_VIDEO=1 for local ffmpeg integration")
	}
	for _, name := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48:rate=12:duration=1", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-pix_fmt", "yuv420p", "-c:v", "libx264", "-c:a", "aac", "-shortest", "-threads", "1", filepath.Join(root, "source.mp4"))
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("source: %v %s", err, b)
	}
	svc, err := New(root, true)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	for _, format := range []string{"gif", "png", "mp4", "webm"} {
		t.Run(format, func(t *testing.T) {
			request := VideoRequest{RenderRequest: RenderRequest{Input: "source.mp4", Output: "result." + format, Palette: "gameboy"}, Frames: 6, FPS: 12}
			request.Options.Width = 64
			if format == "mp4" {
				// H.264 output pads odd dimensions to the next even number.
				request.Options.Width = 63
			}
			b, _ := json.Marshal(request)
			value, err := svc.Do(ctx, "dither_video", b)
			if err != nil {
				t.Fatal(err)
			}
			a := value.(Artifact)
			if a.Frames != 6 || a.Bytes == 0 || a.Operation != "dither_video" {
				t.Fatal(a)
			}
			if _, err = os.Stat(filepath.Join(root, a.Path)); err != nil {
				t.Fatal(err)
			}
			if format == "mp4" || format == "webm" {
				cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-count_frames", "-show_entries", "stream=codec_type,codec_name,width,height,nb_read_frames", "-of", "json", filepath.Join(root, a.Path))
				b, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("probe: %v %s", err, b)
				}
				var probe struct {
					Streams []struct {
						Type   string `json:"codec_type"`
						Codec  string `json:"codec_name"`
						Width  int    `json:"width"`
						Height int    `json:"height"`
						Frames string `json:"nb_read_frames"`
					} `json:"streams"`
				}
				if err := json.Unmarshal(b, &probe); err != nil {
					t.Fatal(err)
				}
				if len(probe.Streams) != 1 || probe.Streams[0].Type != "video" {
					t.Fatalf("output must contain one video stream and no audio: %s", b)
				}
				stream := probe.Streams[0]
				codec := map[string]string{"mp4": "h264", "webm": "vp9"}[format]
				if stream.Codec != codec || stream.Frames != "6" || stream.Width != a.Width || stream.Height != a.Height {
					t.Fatalf("encoded output differs from its metadata: %s. Artifact=%+v", b, a)
				}
				if stream.Width%2 != 0 || stream.Height%2 != 0 {
					t.Fatalf("encoded dimensions must be even: %s", b)
				}
			}
		})
	}
}

func TestVideoInputContainers(t *testing.T) {
	if os.Getenv("DITHER_TEST_VIDEO") != "1" {
		t.Skip("set DITHER_TEST_VIDEO=1 for local ffmpeg integration")
	}
	for _, name := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ name, container, codec string }{
		{"mp4", "mp4", "libx264"},
		{"mov", "mov", "libx264"},
		{"webm", "webm", "libvpx-vp9"},
		{"mkv", "matroska", "ffv1"},
		{"avi", "avi", "mpeg4"},
		{"renamed.bin", "mp4", "libx264"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			input := "source." + tc.name
			cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48:rate=12:duration=0.5", "-pix_fmt", "yuv420p", "-c:v", tc.codec, "-threads", "1", "-f", tc.container, filepath.Join(root, input))
			if b, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("create %s fixture: %v: %s", tc.container, err, b)
			}
			svc, err := New(root, true)
			if err != nil {
				t.Fatal(err)
			}
			defer svc.Close()
			request := VideoRequest{RenderRequest: RenderRequest{Input: input, Output: "result.gif", Palette: "gameboy"}, Frames: 3, FPS: 12}
			request.Options.Width = 64
			data, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			value, err := svc.Do(ctx, "dither_video", data)
			if err != nil {
				t.Fatal(err)
			}
			artifact := value.(Artifact)
			if artifact.Frames != 3 || artifact.Format != "gif" || artifact.Width != 64 || artifact.Height != 48 || artifact.Bytes == 0 {
				t.Fatalf("unexpected result for %s: %+v", tc.container, artifact)
			}
		})
	}
}
