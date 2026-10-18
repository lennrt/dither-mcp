package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const favoriteByteLimit = 64 << 10
const favoriteEntryLimit = 512

type FavoriteSaveRequest struct {
	Name   string `json:"name" jsonschema:"Name this favorite using 1 through 80 Unicode characters after trimming surrounding whitespace. Names are case-sensitive and cannot contain control characters. An existing name is never replaced."`
	Recipe Recipe `json:"recipe" jsonschema:"Save these exact version-1 recipe settings without source or mask paths. Image-mask recipes still require a mask when reused."`
}

func (q *FavoriteSaveRequest) UnmarshalJSON(b []byte) error {
	if !utf8.Valid(b) {
		return errors.New("favorite arguments must use valid UTF-8")
	}
	type plain FavoriteSaveRequest
	var value plain
	if err := StrictJSON(b, &value); err != nil {
		return err
	}
	if err := favoriteRecipeJSON(b); err != nil {
		return err
	}
	if err := favoriteNameJSON(b); err != nil {
		return err
	}
	*q = FavoriteSaveRequest(value)
	return nil
}

// Preserve the exact name across JavaScript/Go: encoding/json replaces lone
// UTF-16 surrogate escapes with U+FFFD. Actual U+FFFD characters remain valid.
func favoriteNameJSON(b []byte) error {
	var fields struct {
		Name json.RawMessage `json:"name"`
	}
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	name := bytes.TrimSpace(fields.Name)
	if len(name) < 2 || name[0] != '"' {
		return errors.New("favorite name must be a Unicode string")
	}
	for i := 1; i < len(name)-1; i++ {
		if name[i] != '\\' {
			continue
		}
		i++
		if name[i] != 'u' {
			continue
		}
		value, err := strconv.ParseUint(string(name[i+1:i+5]), 16, 16)
		if err != nil {
			return err
		}
		i += 4
		if value >= 0xd800 && value <= 0xdbff {
			if i+6 >= len(name) || name[i+1] != '\\' || name[i+2] != 'u' {
				return errors.New("favorite name cannot contain an unpaired Unicode surrogate")
			}
			low, err := strconv.ParseUint(string(name[i+3:i+7]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return errors.New("favorite name cannot contain an unpaired Unicode surrogate")
			}
			i += 6
		} else if value >= 0xdc00 && value <= 0xdfff {
			return errors.New("favorite name cannot contain an unpaired Unicode surrogate")
		}
	}
	return nil
}

// JSON null must not silently become an empty Config. Go's decoder normally
// accepts null for a struct, and replaces malformed UTF-8 inside strings.
func favoriteRecipeJSON(b []byte) error {
	var fields struct {
		Recipe json.RawMessage `json:"recipe"`
	}
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	if len(fields.Recipe) == 0 || bytes.Equal(bytes.TrimSpace(fields.Recipe), []byte("null")) {
		return errors.New("favorite recipe must be a version-1 recipe object")
	}
	var recipe struct {
		Options json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(fields.Recipe, &recipe); err != nil {
		return err
	}
	if value := bytes.TrimSpace(recipe.Options); len(value) == 0 || value[0] != '{' {
		return errors.New("favorite recipe options must be a JSON object")
	}
	return nil
}

type FavoriteLoadRequest struct {
	ID string `json:"id" jsonschema:"Load the opaque 64-character lowercase hexadecimal ID returned by favorite save or favorites list."`
}

type FavoriteSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Favorite struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Recipe Recipe `json:"recipe"`
}

type FavoriteList struct {
	Favorites   []FavoriteSummary `json:"favorites"`
	Unavailable int               `json:"unavailable" jsonschema:"Number of recognized favorite entries that could not be read or validated. Other directory entries are not favorites but count toward the bounded scan."`
}

type favoriteEnvelope struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	Recipe  Recipe `json:"recipe"`
}

func favoriteName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 80 {
		return "", errors.New("favorite name must contain 1 through 80 Unicode characters after trimming whitespace")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", errors.New("favorite name cannot contain control characters")
		}
	}
	return name, nil
}

func validFavoriteID(id string) bool {
	if len(id) != 64 {
		return false
	}
	for _, r := range id {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// Open each directory relative to an already rooted handle. Observable
// symlinks are refused, and os.Root constrains resolution during concurrent
// filesystem changes. No directory is created by a read operation.
func (s *Service) favoritesRoot(create bool) (*os.Root, error) {
	current := s.root
	for _, name := range []string{".dither", "favorites"} {
		st, err := current.Lstat(name)
		if errors.Is(err, os.ErrNotExist) && create {
			if err = current.Mkdir(name, 0755); err == nil || errors.Is(err, os.ErrExist) {
				st, err = current.Lstat(name)
			}
		}
		if err == nil && (!st.IsDir() || st.Mode()&os.ModeSymlink != 0) {
			err = errors.New("favorite storage must use real directories, not symlinks or special files")
		}
		if err != nil {
			if current != s.root {
				current.Close()
			}
			return nil, err
		}
		next, err := current.OpenRoot(name)
		if current != s.root {
			current.Close()
		}
		if err != nil {
			return nil, err
		}
		current = next
	}
	return current, nil
}

func favoriteEntries(ctx context.Context, dir *os.Root) ([]os.DirEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := dir.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(favoriteEntryLimit + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(entries) > favoriteEntryLimit {
		return nil, errors.New("favorite directory exceeds the 512-entry listing limit; load known IDs directly or remove unneeded files from .dither/favorites")
	}
	return entries, nil
}

func readFavorite(ctx context.Context, dir *os.Root, id string) (Favorite, error) {
	if err := ctx.Err(); err != nil {
		return Favorite{}, err
	}
	name := id + ".json"
	st, err := dir.Lstat(name)
	if err != nil {
		return Favorite{}, err
	}
	if !st.Mode().IsRegular() {
		return Favorite{}, errors.New("favorite must be a regular file, not a symlink or special file")
	}
	if st.Size() > favoriteByteLimit {
		return Favorite{}, errors.New("favorite exceeds the 64 KiB size limit")
	}
	f, err := dir.OpenFile(name, os.O_RDONLY|readNoBlockFlag, 0)
	if err != nil {
		return Favorite{}, err
	}
	defer f.Close()
	st, err = f.Stat()
	if err != nil {
		return Favorite{}, err
	}
	if !st.Mode().IsRegular() {
		return Favorite{}, errors.New("favorite must be a regular file")
	}
	b, err := io.ReadAll(io.LimitReader(f, favoriteByteLimit+1))
	if err != nil {
		return Favorite{}, err
	}
	if len(b) > favoriteByteLimit {
		return Favorite{}, errors.New("favorite exceeds the 64 KiB size limit")
	}
	if err := ctx.Err(); err != nil {
		return Favorite{}, err
	}
	if !utf8.Valid(b) {
		return Favorite{}, errors.New("favorite JSON must use valid UTF-8")
	}
	var envelope favoriteEnvelope
	if err := StrictJSON(b, &envelope); err != nil {
		return Favorite{}, fmt.Errorf("invalid favorite: %w", err)
	}
	if err := favoriteRecipeJSON(b); err != nil {
		return Favorite{}, err
	}
	if err := favoriteNameJSON(b); err != nil {
		return Favorite{}, err
	}
	canonical, err := favoriteName(envelope.Name)
	if err != nil || canonical != envelope.Name || envelope.Version != 1 || digest([]byte(envelope.Name)) != id {
		return Favorite{}, errors.New("favorite version, name, or ID does not match its storage envelope")
	}
	if err := validateRecipe(ctx, envelope.Recipe); err != nil {
		return Favorite{}, fmt.Errorf("invalid favorite recipe: %w", err)
	}
	return Favorite{ID: id, Name: envelope.Name, Recipe: envelope.Recipe}, nil
}

func (s *Service) saveFavorite(ctx context.Context, q FavoriteSaveRequest) (Favorite, error) {
	name, err := favoriteName(q.Name)
	if err != nil {
		return Favorite{}, err
	}
	if err := validateRecipe(ctx, q.Recipe); err != nil {
		return Favorite{}, err
	}
	b, err := json.MarshalIndent(favoriteEnvelope{Version: 1, Name: name, Recipe: q.Recipe}, "", "  ")
	if err != nil {
		return Favorite{}, err
	}
	b = append(b, '\n')
	if len(b) > favoriteByteLimit {
		return Favorite{}, errors.New("favorite exceeds the 64 KiB size limit")
	}
	if err := ctx.Err(); err != nil {
		return Favorite{}, err
	}
	dir, err := s.favoritesRoot(true)
	if err != nil {
		return Favorite{}, err
	}
	defer dir.Close()
	entries, err := favoriteEntries(ctx, dir)
	if err != nil {
		return Favorite{}, err
	}
	// This conservative admission check keeps serial saves discoverable. It is
	// not a cross-process capacity lock; atomic publication protects each name.
	if len(entries) >= favoriteEntryLimit {
		return Favorite{}, errors.New("favorite directory has reached its 512-entry listing limit; remove unneeded files from .dither/favorites before saving")
	}
	id := digest([]byte(name))
	if err := publishFile(ctx, dir, id+".json", b); err != nil {
		return Favorite{}, fmt.Errorf("save favorite %q: %w", name, err)
	}
	return Favorite{ID: id, Name: name, Recipe: q.Recipe}, nil
}

func (s *Service) loadFavorite(ctx context.Context, q FavoriteLoadRequest) (Favorite, error) {
	if !validFavoriteID(q.ID) {
		return Favorite{}, errors.New("favorite ID must be exactly 64 lowercase hexadecimal characters")
	}
	if err := ctx.Err(); err != nil {
		return Favorite{}, err
	}
	dir, err := s.favoritesRoot(false)
	if err != nil {
		return Favorite{}, err
	}
	defer dir.Close()
	return readFavorite(ctx, dir, q.ID)
}

func (s *Service) listFavorites(ctx context.Context) (FavoriteList, error) {
	result := FavoriteList{Favorites: []FavoriteSummary{}}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	dir, err := s.favoritesRoot(false)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	defer dir.Close()
	entries, err := favoriteEntries(ctx, dir)
	if err != nil {
		return result, err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return FavoriteList{}, err
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if entry.Name() != id+".json" || !validFavoriteID(id) {
			continue
		}
		favorite, err := readFavorite(ctx, dir, id)
		if err != nil {
			if ctx.Err() != nil {
				return FavoriteList{}, ctx.Err()
			}
			result.Unavailable++
			continue
		}
		result.Favorites = append(result.Favorites, FavoriteSummary{ID: favorite.ID, Name: favorite.Name})
	}
	sort.Slice(result.Favorites, func(i, j int) bool { return result.Favorites[i].Name < result.Favorites[j].Name })
	return result, ctx.Err()
}
