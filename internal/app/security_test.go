package app

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func securityFixture(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	var b bytes.Buffer
	im := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			im.SetNRGBA(x, y, color.NRGBA{uint8(x * 31), uint8(y * 31), 128, 255})
		}
	}
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "input.png"), b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, root
}
func securityCall(s *Service, name string, q any) (any, error) {
	b, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}
	return s.Do(context.Background(), name, b)
}

func TestSecurityRejectsNonlocalPaths(t *testing.T) {
	s, root := securityFixture(t)
	for _, p := range []string{"../input.png", "sub/../../input.png", filepath.Join(root, "input.png"), "https://example.com/image.png", "file:///etc/passwd", "C:\\image.png", "input.png\x00x", ""} {
		t.Run(p, func(t *testing.T) {
			if _, err := securityCall(s, "dither_inspect", InputRequest{Input: p}); err == nil {
				t.Errorf("accepted nonlocal input %q", p)
			}
			if _, err := securityCall(s, "dither_render", RenderRequest{Input: "input.png", Output: p, Format: "png"}); err == nil {
				t.Errorf("accepted nonlocal output %q", p)
			}
		})
	}
}

func TestSecurityRejectsSymlinkEscapes(t *testing.T) {
	s, root := securityFixture(t)
	outside := t.TempDir()
	data, err := os.ReadFile(filepath.Join(root, "input.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(outside, "outside.png"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err = securityCall(s, "dither_inspect", InputRequest{Input: "escape/outside.png"}); err == nil {
		t.Fatal("read followed symlink outside root")
	}
	if _, err = securityCall(s, "dither_render", RenderRequest{Input: "input.png", Output: "escape/written.png"}); err == nil {
		t.Fatal("write followed symlink outside root")
	}
	if _, err = os.Stat(filepath.Join(outside, "written.png")); !os.IsNotExist(err) {
		t.Fatalf("outside output exists or unexpected stat: %v", err)
	}
}

func TestSecurityConcurrentPublicationHasOneWinner(t *testing.T) {
	first, root := securityFixture(t)
	second, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	request := RenderRequest{Input: "input.png", Output: "collision.png"}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, s := range []*Service{first, second} {
		wg.Add(1)
		go func(s *Service) {
			defer wg.Done()
			<-start
			_, err := securityCall(s, "dither_render", request)
			results <- err
		}(s)
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("got %d winners, want exactly 1", successes)
	}
	b, err := os.ReadFile(filepath.Join(root, "collision.png"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = png.Decode(bytes.NewReader(b)); err != nil {
		t.Fatalf("published output is incomplete: %v", err)
	}
	leftovers, err := filepath.Glob(filepath.Join(root, ".dither-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) > 0 {
		t.Fatalf("temporary files leaked: %v", leftovers)
	}
}

func TestSecurityOutputNeverOverwrites(t *testing.T) {
	s, root := securityFixture(t)
	before := []byte("pre-existing user content")
	if err := os.WriteFile(filepath.Join(root, "keep.png"), before, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := securityCall(s, "dither_render", RenderRequest{Input: "input.png", Output: "keep.png"}); err == nil {
		t.Fatal("existing destination accepted")
	}
	after, err := os.ReadFile(filepath.Join(root, "keep.png"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("existing destination changed")
	}
	if _, err = s.Do(context.Background(), "dither_render", []byte(`{"input":"input.png","output":"keep.png","overwrite":true}`)); err == nil {
		t.Fatal("undocumented overwrite flag accepted")
	}
}

func TestSecurityStrictArguments(t *testing.T) {
	s, _ := securityFixture(t)
	cases := []string{`null`, `[]`, `true`, `{"surprise":1}`, `{} {}`, `{"input":1}`, `{"input":"input.png","options":{"unknown":true}}`, strings.Repeat(" ", 1<<20+1)}
	for i, raw := range cases {
		if _, err := s.Do(context.Background(), "dither_render", []byte(raw)); err == nil {
			t.Errorf("case %d accepted invalid arguments", i)
		}
	}
	if _, err := securityCall(s, "not_a_tool", Empty{}); err == nil {
		t.Fatal("unknown tool accepted")
	}
	// Invalid input must not poison a subsequent call in this process.
	if _, err := securityCall(s, "dither_inspect", InputRequest{Input: "input.png"}); err != nil {
		t.Fatalf("server unusable after invalid input: %v", err)
	}
}

func TestSecurityLimitsFailBeforePublication(t *testing.T) {
	s, root := securityFixture(t)
	requests := []struct {
		name string
		q    any
	}{
		{"dither_render", RenderRequest{Input: "input.png", Output: "huge.png", Options: engine.Config{Width: engine.MaxDimension + 1}}},
		{"dither_compare", CompareRequest{RenderRequest: RenderRequest{Input: "input.png", Output: "many.png"}, Algorithms: make([]string, 13)}},
		{"dither_batch", BatchRequest{Items: make([]RenderRequest, 33)}},
		{"dither_animate", AnimateRequest{RenderRequest: RenderRequest{Input: "input.png", Output: "many.gif"}, Frames: MaxFrames + 1}},
	}
	for _, q := range requests {
		if _, err := securityCall(s, q.name, q.q); err == nil {
			t.Errorf("%s accepted excessive request", q.name)
		}
	}
	f, err := os.Create(filepath.Join(root, "oversized.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(MaxBytes + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = securityCall(s, "dither_inspect", InputRequest{Input: "oversized.png"}); err == nil {
		t.Error("oversized source accepted")
	}
	for _, p := range []string{"huge.png", "many.png", "many.gif"} {
		if _, err = os.Stat(filepath.Join(root, p)); !os.IsNotExist(err) {
			t.Errorf("failed request published %s", p)
		}
	}
}

func TestSecurityCancellationDoesNotPublish(t *testing.T) {
	s, root := securityFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	q, _ := json.Marshal(RenderRequest{Input: "input.png", Output: "cancelled.png"})
	if _, err := s.Do(ctx, "dither_render", q); err == nil {
		t.Fatal("cancelled request succeeded")
	}
	if _, err := os.Stat(filepath.Join(root, "cancelled.png")); !os.IsNotExist(err) {
		t.Fatalf("cancelled request published output: %v", err)
	}
}

func TestSecurityRecipeValidation(t *testing.T) {
	s, root := securityFixture(t)
	cases := []Recipe{
		{Version: 2, Options: engine.DefaultConfig()},
		{Version: 1, Options: engine.Config{Algorithm: "not-real"}},
		{Version: 1, Colors: []string{"#not-a-color"}},
		{Version: 1, Options: engine.Config{Gamma: engine.Float(0)}},
		{Version: 1, Options: engine.Config{Crop: &engine.Rect{Width: -1, Height: 4}}},
		{Version: 1, Options: engine.Config{Mask: &engine.Mask{Shape: "not-real"}}},
	}
	for i, r := range cases {
		_, err := securityCall(s, "dither_recipe_save", RecipeSaveRequest{Output: "invalid.json", Recipe: r})
		if err == nil {
			t.Errorf("invalid recipe %d was saved", i)
			os.Remove(filepath.Join(root, "invalid.json"))
		}
	}
}
