package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lennrt/dither-mcp/engine"
)

// Video support requires explicit opt-in. ffmpeg receives a private staging copy,
// forced demuxers, bounded duration and frame dimensions, and generated PNG paths.
// The service runs processes without a shell. User arguments cannot select executable names.
func (s *Service) video(ctx context.Context, q VideoRequest) (Artifact, error) {
	if !s.allowVideo {
		return Artifact{}, errors.New("video disabled: start with --allow-video to enable local ffmpeg/ffprobe")
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return Artifact{}, errors.New("ffmpeg is required for video")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return Artifact{}, errors.New("ffprobe is required for video")
	}
	if q.Frames == 0 {
		q.Frames = 48
	}
	if q.FPS == 0 {
		q.FPS = 12
	}
	if q.Frames < 1 || q.Frames > MaxFrames || q.FPS < 1 || q.FPS > 50 {
		return Artifact{}, errors.New("video frames must be 1..120 and fps 1..50")
	}
	if math.IsNaN(q.Start) || math.IsInf(q.Start, 0) || q.Start < 0 || q.Start > 86400 {
		return Artifact{}, errors.New("start must be in [0,86400] seconds")
	}
	c, r, err := s.config(ctx, q.RenderRequest)
	if err != nil {
		return Artifact{}, err
	}
	if c.Width == 0 && c.Height == 0 {
		c.Width = 480
	}
	if q.DPI != 0 {
		return Artifact{}, errors.New("video does not support dpi metadata")
	}
	input, err := s.read(q.Input, MaxBytes)
	if err != nil {
		return Artifact{}, err
	}
	demux := ""
	switch {
	case len(input) > 12 && string(input[4:8]) == "ftyp":
		demux = "mov"
	case len(input) > 4 && bytes.Equal(input[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}):
		demux = "matroska"
	case len(input) > 12 && string(input[:4]) == "RIFF" && string(input[8:12]) == "AVI ":
		demux = "avi"
	default:
		return Artifact{}, errors.New("video input must be an MP4/MOV with ftyp header, WebM/Matroska, or AVI container")
	}
	staging, err := os.MkdirTemp("", "dither-video-")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(staging)
	if err = os.WriteFile(filepath.Join(staging, "input.media"), input, 0600); err != nil {
		return Artifact{}, err
	}
	movOptions := []string{}
	if demux == "mov" {
		movOptions = []string{"-enable_drefs", "0", "-use_absolute_path", "0"}
	}
	probe := &boundedBuffer{max: 64 << 10}
	stderr := &boundedBuffer{max: 64 << 10}
	probeArgs := []string{"-v", "error", "-max_alloc", "268435456", "-max_pixels", strconv.Itoa(engine.MaxPixels), "-protocol_whitelist", "file,pipe", "-f", demux}
	probeArgs = append(probeArgs, movOptions...)
	probeArgs = append(probeArgs, "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "json", "input.media")
	cmd := exec.CommandContext(ctx, ffprobe, probeArgs...)
	cmd.Dir = staging
	cmd.Stdout = probe
	cmd.Stderr = stderr
	if err = cmd.Run(); err != nil {
		return Artifact{}, fmt.Errorf("ffprobe: %w: %s", err, stderr.Bytes())
	}
	var meta struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	if err = json.Unmarshal(probe.Bytes(), &meta); err != nil || len(meta.Streams) != 1 {
		return Artifact{}, errors.New("video must contain a readable video stream")
	}
	w, h := meta.Streams[0].Width, meta.Streams[0].Height
	if w <= 0 || h <= 0 || w > engine.MaxDimension || h > engine.MaxDimension || int64(w)*int64(h) > engine.MaxPixels {
		return Artifact{}, errors.New("video dimensions exceed limits")
	}
	if _, _, err := frameDimensions(image.Rect(0, 0, w, h), c, q.Frames); err != nil {
		return Artifact{}, err
	}
	// Decode only bounded native-size frames. The engine applies crop and resize
	// exactly as it does for still images. The disk budget follows the input frame budget.
	if int64(w)*int64(h)*int64(q.Frames) > MaxFramePixels {
		return Artifact{}, errors.New("decoded video exceeds frame-pixel budget. Reduce the frame count")
	}
	args := []string{"-nostdin", "-v", "error", "-max_pixels", strconv.Itoa(engine.MaxPixels), "-threads", "1", "-protocol_whitelist", "file,pipe", "-f", demux, "-ss", strconv.FormatFloat(q.Start, 'f', 3, 64), "-i", "input.media", "-map", "0:v:0", "-an", "-sn", "-dn", "-t", strconv.FormatFloat(float64(q.Frames)/float64(q.FPS), 'f', 4, 64), "-vf", fmt.Sprintf("fps=%d", q.FPS), "-frames:v", strconv.Itoa(q.Frames), "-threads", "1", "frame-%04d.png"}
	if len(movOptions) > 0 {
		at := 0
		for i, a := range args {
			if a == "-i" {
				at = i
				break
			}
		}
		joined := append([]string{}, args[:at]...)
		joined = append(joined, movOptions...)
		args = append(joined, args[at:]...)
	}
	if err = runMedia(ctx, ffmpeg, staging, args); err != nil {
		return Artifact{}, err
	}
	paths, err := filepath.Glob(filepath.Join(staging, "frame-*.png"))
	if err != nil || len(paths) == 0 {
		return Artifact{}, errors.New("no frames decoded at requested start")
	}
	frames := make([]*image.NRGBA, 0, len(paths))
	pixels := int64(0)
	for i, p := range paths {
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		st, err := os.Stat(p)
		if err != nil {
			return Artifact{}, err
		}
		if st.Size() > MaxBytes {
			return Artifact{}, errors.New("decoded frame exceeds byte limit")
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return Artifact{}, err
		}
		im, _, err := decodeBytes(ctx, b)
		if err != nil {
			return Artifact{}, err
		}
		out, err := engine.Process(ctx, im, c)
		if err != nil {
			return Artifact{}, err
		}
		pixels += int64(out.Bounds().Dx()) * int64(out.Bounds().Dy())
		if pixels > MaxFramePixels {
			return Artifact{}, errors.New("rendered video exceeds frame-pixel budget")
		}
		frames = append(frames, out)
		encoded, err := encodeBounded("png", out, 0)
		if err != nil {
			return Artifact{}, err
		}
		if err = os.WriteFile(filepath.Join(staging, fmt.Sprintf("out-%04d.png", i+1)), encoded, 0600); err != nil {
			return Artifact{}, err
		}
	}
	format := strings.ToLower(q.Format)
	if format == "" {
		format = strings.ToLower(stringsExt(q.Output))
	}
	if format == "gif" || format == "png" {
		return s.publishFrames(ctx, q.Output, format, frames, nil, q.FPS, 0, 6, &r)
	}
	if format != "mp4" && format != "webm" {
		return Artifact{}, errors.New("video output must be mp4, webm, gif or png spritesheet")
	}
	args = []string{"-nostdin", "-v", "error", "-threads", "1", "-framerate", strconv.Itoa(q.FPS), "-i", "out-%04d.png", "-frames:v", strconv.Itoa(len(frames)), "-an", "-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2", "-pix_fmt", "yuv420p", "-threads", "1"}
	if format == "mp4" {
		args = append(args, "-c:v", "libx264", "-crf", "18", "-movflags", "+faststart")
	} else {
		args = append(args, "-c:v", "libvpx-vp9", "-crf", "24", "-b:v", "0")
	}
	args = append(args, "-fs", strconv.FormatInt(MaxBytes+1, 10), "result."+format)
	if err = runMedia(ctx, ffmpeg, staging, args); err != nil {
		return Artifact{}, err
	}
	p := filepath.Join(staging, "result."+format)
	st, err := os.Stat(p)
	if err != nil {
		return Artifact{}, err
	}
	if st.Size() > MaxBytes {
		return Artifact{}, errors.New("encoded video exceeds byte limit")
	}
	// A byte-limited encoder can exit successfully after truncating a clip.
	// Verify the encoded frame count before publishing any video artifact.
	probe = &boundedBuffer{max: 64 << 10}
	stderr = &boundedBuffer{max: 64 << 10}
	check := exec.CommandContext(ctx, ffprobe, "-v", "error", "-max_alloc", "268435456", "-count_frames", "-select_streams", "v:0", "-show_entries", "stream=nb_read_frames", "-of", "default=noprint_wrappers=1:nokey=1", "result."+format)
	check.Dir = staging
	check.Stdout = probe
	check.Stderr = stderr
	if err = check.Run(); err != nil {
		return Artifact{}, fmt.Errorf("verify encoded video: %w", err)
	}
	actual, err := strconv.Atoi(strings.TrimSpace(string(probe.Bytes())))
	if err != nil || actual != len(frames) {
		return Artifact{}, errors.New("encoded video frame count mismatch (possible byte-limit truncation)")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return Artifact{}, err
	}
	width, height := frames[0].Bounds().Dx(), frames[0].Bounds().Dy()
	return s.publish(ctx, q.Output, format, b, width+width%2, height+height%2, len(frames), &r)
}
func runMedia(ctx context.Context, exe, dir string, args []string) error {
	stderr := &boundedBuffer{max: 64 << 10}
	args = append([]string{"-max_alloc", "268435456"}, args...)
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = dir
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, stderr.Bytes())
	}
	return nil
}
