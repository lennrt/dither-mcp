package app

import (
	"context"
	"encoding/json"
	"github.com/lennrt/dither-mcp/engine"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type cancelAfterPublication struct {
	context.Context
	path string
}

func (c cancelAfterPublication) Err() error {
	if _, err := os.Stat(c.path); err == nil {
		return context.Canceled
	}
	return nil
}

func TestBatchReportsEveryItemAfterCancellation(t *testing.T) {
	root := t.TempDir()
	f, err := os.Create(filepath.Join(root, "input.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	q := BatchRequest{Items: []RenderRequest{{Input: "input.png", Output: "first.png"}, {Input: "input.png", Output: "second.png"}, {Input: "input.png", Output: "third.png"}}}
	result, err := s.batch(cancelAfterPublication{context.Background(), filepath.Join(root, "first.png")}, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("got %d outcomes, want 3", len(result.Items))
	}
	if result.Items[0].Artifact == nil || result.Items[0].Error != "" {
		t.Fatalf("first success lost: %+v", result.Items[0])
	}
	for i := 1; i < 3; i++ {
		item := result.Items[i]
		if item.Index != i || item.Artifact != nil || item.Error != context.Canceled.Error() {
			t.Fatalf("invalid canceled outcome: %+v", item)
		}
		if _, err := os.Stat(filepath.Join(root, q.Items[i].Output)); !os.IsNotExist(err) {
			t.Fatalf("canceled item published: %v", err)
		}
	}
}

func TestBatchCancellationResultsReachServiceCaller(t *testing.T) {
	root := t.TempDir()
	f, err := os.Create(filepath.Join(root, "input.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observed := make(chan struct{})
	go func() {
		defer close(observed)
		for {
			if _, err := os.Stat(filepath.Join(root, "first.png")); err == nil {
				cancel()
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
				runtime.Gosched()
			}
		}
	}()
	q := BatchRequest{Items: []RenderRequest{{Input: "input.png", Output: "first.png"}, {Input: "input.png", Output: "second.png", Options: engine.Config{Width: 2048, Height: 2048}}, {Input: "input.png", Output: "third.png"}}}
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	value, err := s.Do(ctx, "dither_batch", raw)
	if err != nil {
		t.Fatal(err)
	}
	<-observed
	result, ok := value.(BatchResult)
	if !ok {
		t.Fatalf("wrong result type: %T", value)
	}
	if len(result.Items) != 3 {
		t.Fatalf("service returned %d outcomes, want 3", len(result.Items))
	}
	if result.Items[0].Artifact == nil || result.Items[0].Artifact.Operation != "dither_render" {
		t.Fatal("committed artifact/provenance lost")
	}
	for i := 1; i < 3; i++ {
		if result.Items[i].Index != i || result.Items[i].Artifact != nil || result.Items[i].Error != context.Canceled.Error() {
			t.Fatalf("wrong canceled outcome: %+v", result.Items[i])
		}
	}
}
