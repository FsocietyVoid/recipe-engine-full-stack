package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"recipe-scraper/internal/models"
)

// Store handles persistence of scraped recipes.
type Store struct {
	dir string
	mu  sync.Mutex
}

// New creates a Store backed by the given directory.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// SaveRecipe writes a single recipe to <dir>/<id>.json.
func (s *Store) SaveRecipe(r models.Recipe) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := sanitiseFilename(r.ID)
	if id == "" {
		id = fmt.Sprintf("recipe_%d", time.Now().UnixNano())
	}
	path := filepath.Join(s.dir, id+".json")

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file %s: %w", path, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// SaveAll writes every recipe in a single recipes.json file.
func (s *Store) SaveAll(recipes []models.Recipe) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, "recipes.json")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create recipes.json: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(recipes)
}

// LoadAll loads all recipes from recipes.json.
func (s *Store) LoadAll() ([]models.Recipe, error) {
	path := filepath.Join(s.dir, "recipes.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open recipes.json: %w", err)
	}
	defer f.Close()

	var recipes []models.Recipe
	if err := json.NewDecoder(f).Decode(&recipes); err != nil {
		return nil, fmt.Errorf("decode recipes.json: %w", err)
	}
	return recipes, nil
}

func sanitiseFilename(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
