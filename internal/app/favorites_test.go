package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/lennrt/dither-mcp/engine"
)

func TestFavoritesPersistExactRecipeAcrossServices(t *testing.T) {
	s, root := studioFixture(t, 32, 24)
	preview := studioCall(t, s, StudioRequest{Input: "source.png", MaskInput: "source.png", Colors: []string{"#172a35", "#fae8d2"}, Options: engine.Config{Algorithm: "blue-noise", Width: 16, Seed: 17, Saturation: engine.Float(0), Crop: &engine.Rect{X: 2, Y: 2, Width: 24, Height: 18}, Effects: engine.Effects{Scanlines: .15}}})
	before, _ := json.Marshal(preview.Recipe)
	saved := normCall(t, s, "dither_favorite_save", FavoriteSaveRequest{Name: "  Café moon / 夜  ", Recipe: preview.Recipe}).(Favorite)
	if saved.Name != "Café moon / 夜" || saved.ID != digest([]byte(saved.Name)) || !reflect.DeepEqual(saved.Recipe, preview.Recipe) {
		t.Fatalf("favorite lost exact name or recipe: %+v", saved)
	}
	storePath := filepath.Join(root, ".dither", "favorites", saved.ID+".json")
	stored, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"source.png", "mask_input", "input", "output", "data:image"} {
		if bytes.Contains(stored, []byte(forbidden)) {
			t.Fatalf("favorite contains source, output or pixels: %s", forbidden)
		}
	}
	if s.Close() != nil {
		t.Fatal("closing original service failed")
	}
	fresh, err := New(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	listed := normCall(t, fresh, "dither_favorites_list", Empty{}).(FavoriteList)
	if !reflect.DeepEqual(listed.Favorites, []FavoriteSummary{{ID: saved.ID, Name: saved.Name}}) || listed.Unavailable != 0 {
		t.Fatalf("restart did not discover favorite: %+v", listed)
	}
	loaded := normCall(t, fresh, "dither_favorite_load", FavoriteLoadRequest{ID: saved.ID}).(Favorite)
	if !reflect.DeepEqual(loaded, saved) {
		t.Fatal("fresh service changed stored favorite")
	}
	request := StudioRequest{Input: "source.png", MaskInput: "source.png", Palette: loaded.Recipe.Palette, Colors: loaded.Recipe.Colors, Options: loaded.Recipe.Options}
	if replay := studioCall(t, fresh, request); replay.Data != preview.Data || !reflect.DeepEqual(replay.Recipe, preview.Recipe) {
		t.Fatal("favorite did not replay exact masked preview pixels and settings")
	}
	request.MaskInput = ""
	raw, _ := json.Marshal(request)
	if _, err := fresh.Do(context.Background(), "dither_studio", raw); err == nil {
		t.Fatal("favorite silently dropped the image-mask requirement")
	}
	after, _ := json.Marshal(preview.Recipe)
	if !bytes.Equal(before, after) {
		t.Fatal("saving modified the supplied recipe")
	}
	if _, err := fresh.saveFavorite(context.Background(), FavoriteSaveRequest{Name: saved.Name, Recipe: Recipe{Version: 1}}); !errors.Is(err, os.ErrExist) {
		t.Fatalf("duplicate name was not rejected as existing: %v", err)
	}
	if after, _ := os.ReadFile(storePath); !bytes.Equal(after, stored) {
		t.Fatal("duplicate name replaced original favorite")
	}
}

func TestFavoritesEmptyReadsAndNameSemantics(t *testing.T) {
	s, root := randomStyleService(t)
	listed := normCall(t, s, "dither_favorites_list", Empty{}).(FavoriteList)
	if listed.Favorites == nil || len(listed.Favorites) != 0 || listed.Unavailable != 0 {
		t.Fatalf("missing library did not return a stable empty list: %+v", listed)
	}
	if files, _ := os.ReadDir(root); len(files) != 0 {
		t.Fatal("listing created the library")
	}
	for _, name := range []string{"zebra", "Alpha", "alpha", "é", "e\u0301", strings.Repeat("夜", 80)} {
		fav := normCall(t, s, "dither_favorite_save", FavoriteSaveRequest{Name: name, Recipe: Recipe{Version: 1}}).(Favorite)
		if fav.Name != name || fav.ID != digest([]byte(name)) {
			t.Fatalf("case or Unicode spelling changed: %+v", fav)
		}
	}
	listed = normCall(t, s, "dither_favorites_list", Empty{}).(FavoriteList)
	if len(listed.Favorites) != 6 || listed.Favorites[0].Name != "Alpha" || listed.Favorites[1].Name != "alpha" {
		t.Fatalf("favorites are not distinct and sorted by name: %+v", listed)
	}
}

func TestFavoritesRejectInvalidArgumentsBeforeWriting(t *testing.T) {
	s, root := randomStyleService(t)
	for _, step := range []struct{ tool, raw string }{
		{"dither_favorites_list", `null`}, {"dither_favorites_list", `{"input":"other"}`},
		{"dither_favorite_save", `null`}, {"dither_favorite_save", `[]`},
		{"dither_favorite_save", `{"name":null,"recipe":{"version":1,"options":{}}}`},
		{"dither_favorite_save", `{"name":"   ","recipe":{"version":1,"options":{}}}`},
		{"dither_favorite_save", `{"name":"bad\u0000name","recipe":{"version":1,"options":{}}}`},
		{"dither_favorite_save", `{"name":"bad\nname","recipe":{"version":1,"options":{}}}`},
		{"dither_favorite_save", `{"name":"\ud800","recipe":{"version":1,"options":{}}}`},
		{"dither_favorite_save", `{"name":"\udfff","recipe":{"version":1,"options":{}}}`},
		{"dither_favorite_save", `{"name":"\ud800\u0041","recipe":{"version":1,"options":{}}}`},
		{"dither_favorite_save", `{"name":"test","recipe":null}`},
		{"dither_favorite_save", `{"name":"test","recipe":{"version":1,"options":null}}`},
		{"dither_favorite_save", `{"name":"test","recipe":{"version":1}}`},
		{"dither_favorite_save", "{\"name\":\"bad" + string([]byte{0xff}) + "\",\"recipe\":{\"version\":1,\"options\":{}}}"},
		{"dither_favorite_save", `{"name":"test","recipe":{"version":2,"options":{}}}`},
		{"dither_favorite_save", `{"name":"test","recipe":{"version":1,"input":"source.png","options":{}}}`},
		{"dither_favorite_save", `{"name":"test","recipe":{"version":1,"options":{}},"output":"escape.json"}`},
		{"dither_favorite_save", `{"name":"test","recipe":{"version":1,"options":{"algorithm":"missing"}}}`},
		{"dither_favorite_load", `null`}, {"dither_favorite_load", `{"id":null}`},
		{"dither_favorite_load", `{"id":"../escape"}`}, {"dither_favorite_load", `{"name":"test"}`},
		{"dither_favorite_load", `{"id":"` + strings.Repeat("A", 64) + `"}`},
		{"dither_favorite_save", `{"name":"` + strings.Repeat("夜", 81) + `","recipe":{"version":1,"options":{}}}`},
	} {
		if result, err := s.Do(context.Background(), step.tool, []byte(step.raw)); err == nil || result != nil {
			t.Errorf("accepted %s %s: %+v %v", step.tool, step.raw, result, err)
		}
	}
	if _, err := favoriteName(string([]byte{0xff})); err == nil {
		t.Error("accepted non-UTF8 name")
	}
	// Hex parsing permits surrounding whitespace; preserving that exact input
	// must still respect the on-disk envelope bound before creating a library.
	oversized := FavoriteSaveRequest{Name: "Too large", Recipe: Recipe{Version: 1, Colors: []string{strings.Repeat(" ", favoriteByteLimit) + "#000", "#fff"}}}
	if _, err := s.saveFavorite(context.Background(), oversized); err == nil || !strings.Contains(err.Error(), "64 KiB") {
		t.Fatalf("oversized valid recipe did not fail at storage bound: %v", err)
	}
	if _, err := s.loadFavorite(context.Background(), FavoriteLoadRequest{ID: strings.Repeat("a", 64)}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing known-ID favorite should report not found: %v", err)
	}
	if files, _ := os.ReadDir(root); len(files) != 0 {
		t.Fatal("invalid requests created library files")
	}
}

func TestFavoritesPreserveUnicodePairsAndReplacementCharacter(t *testing.T) {
	s, root := randomStyleService(t)
	for _, item := range []struct{ encoded, want string }{
		{`"\ud83c\udf19"`, "🌙"}, {`"�"`, "�"}, {`"\\ud800"`, `\ud800`},
	} {
		raw := []byte(`{"name":` + item.encoded + `,"recipe":{"version":1,"options":{}}}`)
		result, err := s.Do(context.Background(), "dither_favorite_save", raw)
		if err != nil {
			t.Fatal(err)
		}
		favorite := result.(Favorite)
		if favorite.Name != item.want || favorite.ID != digest([]byte(item.want)) {
			t.Fatalf("changed exact Unicode name: %+v", favorite)
		}
	}
	// A corrupt envelope cannot pass name/hash validation by decoding a lone
	// surrogate into the otherwise valid replacement-character favorite name.
	id := digest([]byte("�"))
	path := filepath.Join(root, ".dither", "favorites", id+".json")
	if err := os.WriteFile(path, []byte(`{"version":1,"name":"\ud800","recipe":{"version":1,"options":{}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.loadFavorite(context.Background(), FavoriteLoadRequest{ID: id}); err == nil || !strings.Contains(err.Error(), "surrogate") {
		t.Fatalf("accepted replaced surrogate envelope: %v", err)
	}
}

func TestFavoritesConcurrentSameNameHasOneCompleteWinner(t *testing.T) {
	root := t.TempDir()
	const writers = 8
	services := make([]*Service, writers)
	for i := range services {
		var err error
		services[i], err = New(root, false)
		if err != nil {
			t.Fatal(err)
		}
		defer services[i].Close()
	}
	type outcome struct {
		favorite Favorite
		err      error
	}
	results := make(chan outcome, writers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, service := range services {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			favorite, err := service.saveFavorite(context.Background(), FavoriteSaveRequest{Name: "One winner", Recipe: Recipe{Version: 1, Options: engine.Config{Seed: int64(i)}}})
			results <- outcome{favorite, err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	var winner Favorite
	wins := 0
	for result := range results {
		if result.err == nil {
			wins++
			winner = result.favorite
		} else if !errors.Is(result.err, os.ErrExist) {
			t.Errorf("unexpected losing writer error: %v", result.err)
		}
	}
	if wins != 1 {
		t.Fatalf("got %d publication winners", wins)
	}
	loaded, err := services[0].loadFavorite(context.Background(), FavoriteLoadRequest{ID: winner.ID})
	if err != nil || !reflect.DeepEqual(loaded, winner) {
		t.Fatalf("published file is partial or belongs to another writer: %+v %v", loaded, err)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".dither", "favorites"))
	if err != nil || len(entries) != 1 || entries[0].Name() != winner.ID+".json" {
		t.Fatalf("publication left temporary or duplicate files: %+v %v", entries, err)
	}
}

func TestFavoritesCorruptEntriesAreBoundedAndUnavailable(t *testing.T) {
	s, root := randomStyleService(t)
	valid := normCall(t, s, "dither_favorite_save", FavoriteSaveRequest{Name: "Valid", Recipe: Recipe{Version: 1}}).(Favorite)
	dir := filepath.Join(root, ".dither", "favorites")
	cases := map[string][]byte{
		"broken":       []byte(`{`),
		"oversized":    bytes.Repeat([]byte(" "), favoriteByteLimit+1),
		"unknown":      []byte(`{"version":1,"name":"unknown","recipe":{"version":1,"options":{}},"extra":true}`),
		"hash":         []byte(`{"version":1,"name":"different","recipe":{"version":1,"options":{}}}`),
		"version":      []byte(`{"version":2,"name":"version","recipe":{"version":1,"options":{}}}`),
		"recipe":       []byte(`{"version":1,"name":"recipe","recipe":{"version":1,"options":{"algorithm":"missing"}}}`),
		"null":         []byte(`null`),
		"null-options": []byte(`{"version":1,"name":"null-options","recipe":{"version":1,"options":null}}`),
		"utf8":         []byte("{\"version\":1,\"name\":\"bad" + string([]byte{0xff}) + "\",\"recipe\":{\"version\":1,\"options\":{}}}"),
	}
	for name, body := range cases {
		id := digest([]byte(name))
		if err := os.WriteFile(filepath.Join(dir, id+".json"), body, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.loadFavorite(context.Background(), FavoriteLoadRequest{ID: id}); err == nil {
			t.Errorf("loaded corrupt %s", name)
		}
	}
	for _, name := range []string{"subdirectory", "symlink"} {
		id := digest([]byte(name))
		target := filepath.Join(dir, id+".json")
		var err error
		if name == "subdirectory" {
			err = os.Mkdir(target, 0700)
		} else {
			err = os.Symlink(valid.ID+".json", target)
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.loadFavorite(context.Background(), FavoriteLoadRequest{ID: id}); err == nil {
			t.Errorf("loaded %s", name)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("unrelated"), 0600); err != nil {
		t.Fatal(err)
	}
	listed := normCall(t, s, "dither_favorites_list", Empty{}).(FavoriteList)
	if len(listed.Favorites) != 1 || listed.Favorites[0].ID != valid.ID || listed.Unavailable != len(cases)+2 {
		t.Fatalf("corruption poisoned valid discovery or was hidden: %+v", listed)
	}
}

func TestFavoritesRefuseStorageSymlinks(t *testing.T) {
	for _, level := range []string{".dither", ".dither/favorites"} {
		t.Run(level, func(t *testing.T) {
			s, root := randomStyleService(t)
			if level != ".dither" {
				if err := os.Mkdir(filepath.Join(root, ".dither"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			outside := t.TempDir()
			if err := os.Symlink(outside, filepath.Join(root, level)); err != nil {
				t.Fatal(err)
			}
			if _, err := s.listFavorites(context.Background()); err == nil {
				t.Fatal("listed symlink storage")
			}
			if _, err := s.saveFavorite(context.Background(), FavoriteSaveRequest{Name: "test", Recipe: Recipe{Version: 1}}); err == nil {
				t.Fatal("saved through symlink storage")
			}
			if entries, _ := os.ReadDir(outside); len(entries) != 0 {
				t.Fatal("favorite escaped workspace")
			}
		})
	}
}

func TestFavoritesEntryLimitPreservesDirectLoad(t *testing.T) {
	s, root := randomStyleService(t)
	valid := normCall(t, s, "dither_favorite_save", FavoriteSaveRequest{Name: "Kept", Recipe: Recipe{Version: 1}}).(Favorite)
	dir := filepath.Join(root, ".dither", "favorites")
	for i := 1; i < favoriteEntryLimit; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("unrelated-%03d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if listed, err := s.listFavorites(context.Background()); err != nil || len(listed.Favorites) != 1 {
		t.Fatalf("exactly512 entries rejected: %+v %v", listed, err)
	}
	if _, err := s.saveFavorite(context.Background(), FavoriteSaveRequest{Name: "Too many", Recipe: Recipe{Version: 1}}); err == nil || !strings.Contains(err.Error(), "512") {
		t.Fatalf("serial save exceeded scan capacity: %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != favoriteEntryLimit {
		t.Fatal("rejected capacity save left a temporary file")
	}
	if err := os.WriteFile(filepath.Join(dir, "external-extra"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.listFavorites(context.Background()); err == nil || !strings.Contains(err.Error(), "known IDs") {
		t.Fatalf("oversized discovery did not report direct-load recovery: %v", err)
	}
	if loaded, err := s.loadFavorite(context.Background(), FavoriteLoadRequest{ID: valid.ID}); err != nil || !reflect.DeepEqual(loaded, valid) {
		t.Fatalf("overfull directory blocked direct load: %+v %v", loaded, err)
	}
}

type favoriteCancelContext struct {
	context.Context
	checks int
}

func (c *favoriteCancelContext) Err() error {
	c.checks--
	if c.checks <= 0 {
		return context.Canceled
	}
	return nil
}

func TestFavoritesCancellationBeforePublicationAndDuringDiscovery(t *testing.T) {
	s, root := randomStyleService(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.saveFavorite(cancelled, FavoriteSaveRequest{Name: "No file", Recipe: Recipe{Version: 1}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancelled save: %v", err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatal("pre-cancelled save created directories")
	}
	// Cancel at the final publication check, after the temporary file was synced.
	ctx := &favoriteCancelContext{Context: context.Background(), checks: 5}
	if _, err := s.saveFavorite(ctx, FavoriteSaveRequest{Name: "No publication", Recipe: Recipe{Version: 1}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("late-cancelled save: %v", err)
	}
	dir := filepath.Join(root, ".dither", "favorites")
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatal("cancelled publication left temporary or final file")
	}
	normCall(t, s, "dither_favorite_save", FavoriteSaveRequest{Name: "Kept", Recipe: Recipe{Version: 1}})
	ctx = &favoriteCancelContext{Context: context.Background(), checks: 4}
	if _, err := s.listFavorites(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("discovery ignored cancellation: %v", err)
	}
}
