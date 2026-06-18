package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/semi710/nastebin/internal/config"
	"github.com/semi710/nastebin/internal/models"
	"github.com/semi710/nastebin/internal/utils"
)

// Store handles paste persistence on the local filesystem.
type Store struct {
	cfg *config.Config
	mu  sync.Mutex
}

// NewStore initializes a Store, ensuring required directories exist.
func NewStore(cfg *config.Config) (*Store, error) {
	s := &Store{cfg: cfg}
	for _, dir := range []string{"public", "private", "metadata"} {
		path := filepath.Join(cfg.DataDir, dir)
		if err := os.MkdirAll(path, 0750); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", path, err)
		}
	}
	return s, nil
}

// dirPath returns the storage directory (public or private) for a paste.
func (s *Store) dirPath(private bool) string {
	if private {
		return filepath.Join(s.cfg.DataDir, "private")
	}
	return filepath.Join(s.cfg.DataDir, "public")
}

// metaPath returns the metadata file path for a paste.
func (s *Store) metaPath(slug string) string {
	return filepath.Join(s.cfg.DataDir, "metadata", slug+".json")
}

// validateSlug ensures a slug is safe to use in filesystem paths.
func validateSlug(slug string) error {
	if err := utils.Validate(slug); err != nil {
		return err
	}
	if strings.Contains(slug, "/") || strings.Contains(slug, "\\") || strings.Contains(slug, "..") {
		return &utils.InvalidSlugError{Reason: "chars", Message: "invalid slug"}
	}
	return nil
}

// Exists checks whether a paste (public or private) already exists.
func (s *Store) Exists(slug string) bool {
	if err := validateSlug(slug); err != nil {
		return false
	}
	for _, private := range []bool{false, true} {
		path := filepath.Join(s.dirPath(private), slug)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	if _, err := os.Stat(s.metaPath(slug)); err == nil {
		return true
	}
	return false
}

// Get retrieves a paste's metadata.
func (s *Store) Get(slug string) (*models.Paste, error) {
	if err := validateSlug(slug); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.metaPath(slug))
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	var p models.Paste
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return &p, nil
}

// GetContent returns the raw content reader for a paste.
func (s *Store) GetContent(slug string, private bool) (io.ReadCloser, error) {
	if err := validateSlug(slug); err != nil {
		return nil, err
	}
	path := filepath.Join(s.dirPath(private), slug)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open paste: %w", err)
	}
	return f, nil
}

// Save writes a new paste atomically.
func (s *Store) Save(p *models.Paste, content io.Reader) error {
	if err := validateSlug(p.Slug); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Write content atomically
	dataDir := s.dirPath(p.Private)
	tmpFile, err := os.CreateTemp(dataDir, ".paste-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	n, err := io.Copy(tmpFile, content)
	if closeErr := tmpFile.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write content: %w", err)
	}

	finalPath := filepath.Join(dataDir, p.Slug)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	// Write metadata
	p.Size = n
	metaData, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	metaTmp, err := os.CreateTemp(filepath.Join(s.cfg.DataDir, "metadata"), ".meta-*")
	if err != nil {
		return fmt.Errorf("create meta temp: %w", err)
	}
	metaTmpPath := metaTmp.Name()
	defer func() { _ = os.Remove(metaTmpPath) }()

	if _, err := metaTmp.Write(metaData); err != nil {
		_ = metaTmp.Close()
		return fmt.Errorf("write metadata: %w", err)
	}
	if err := metaTmp.Close(); err != nil {
		return fmt.Errorf("close meta temp: %w", err)
	}

	metaFinal := s.metaPath(p.Slug)
	if err := os.Rename(metaTmpPath, metaFinal); err != nil {
		return fmt.Errorf("rename metadata: %w", err)
	}

	return nil
}

// Overwrite replaces an existing paste's content.
func (s *Store) Overwrite(slug string, content io.Reader) error {
	if err := validateSlug(slug); err != nil {
		return err
	}

	p, err := s.Get(slug)
	if err != nil {
		return fmt.Errorf("get paste: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Same atomic write logic
	dataDir := s.dirPath(p.Private)
	tmpFile, err := os.CreateTemp(dataDir, ".paste-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	n, err := io.Copy(tmpFile, content)
	if closeErr := tmpFile.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write content: %w", err)
	}

	finalPath := filepath.Join(dataDir, slug)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	// Update metadata size
	p.Size = n
	metaData, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	metaTmp, err := os.CreateTemp(filepath.Join(s.cfg.DataDir, "metadata"), ".meta-*")
	if err != nil {
		return fmt.Errorf("create meta temp: %w", err)
	}
	metaTmpPath := metaTmp.Name()
	defer func() { _ = os.Remove(metaTmpPath) }()

	if _, err := metaTmp.Write(metaData); err != nil {
		_ = metaTmp.Close()
		return fmt.Errorf("write metadata: %w", err)
	}
	if err := metaTmp.Close(); err != nil {
		return fmt.Errorf("close meta temp: %w", err)
	}

	metaFinal := s.metaPath(slug)
	if err := os.Rename(metaTmpPath, metaFinal); err != nil {
		return fmt.Errorf("rename metadata: %w", err)
	}

	return nil
}

// UniqueSlug generates a random slug that does not already exist.
func (s *Store) UniqueSlug() string {
	for {
		slug := utils.Generate()
		if !s.Exists(slug) {
			return slug
		}
	}
}
