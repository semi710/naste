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

// Scope identifies whether a paste is public or private.
type Scope string

const (
	ScopePublic  Scope = "public"
	ScopePrivate Scope = "private"
)

// scopeFor converts a bool (from models.Paste.Private) to a Scope.
func scopeFor(private bool) Scope {
	if private {
		return ScopePrivate
	}
	return ScopePublic
}

// Store handles paste persistence on the local filesystem.
type Store struct {
	cfg *config.Config
	mu  sync.Mutex
}

// NewStore initializes a Store, ensuring required directories exist.
func NewStore(cfg *config.Config) (*Store, error) {
	s := &Store{cfg: cfg}
	for _, dir := range []string{
		string(ScopePublic),
		string(ScopePrivate),
		"metadata/" + string(ScopePublic),
		"metadata/" + string(ScopePrivate),
	} {
		path := filepath.Join(cfg.DataDir, dir)
		if err := os.MkdirAll(path, 0750); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", path, err)
		}
	}
	return s, nil
}

func (s *Store) contentPath(slug string, scope Scope) string {
	return filepath.Join(s.cfg.DataDir, string(scope), slug)
}

func (s *Store) metaPath(slug string, scope Scope) string {
	return filepath.Join(s.cfg.DataDir, "metadata", string(scope), slug+".json")
}

func (s *Store) metaDir(scope Scope) string {
	return filepath.Join(s.cfg.DataDir, "metadata", string(scope))
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

// Exists checks whether a paste with the given slug exists in the given scope.
func (s *Store) Exists(slug string, scope Scope) bool {
	if err := validateSlug(slug); err != nil {
		return false
	}
	_, err := os.Stat(s.contentPath(slug, scope))
	return err == nil
}

// Get retrieves a paste's metadata from the given scope.
func (s *Store) Get(slug string, scope Scope) (*models.Paste, error) {
	if err := validateSlug(slug); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.metaPath(slug, scope))
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
func (s *Store) GetContent(slug string, scope Scope) (io.ReadCloser, error) {
	if err := validateSlug(slug); err != nil {
		return nil, err
	}
	f, err := os.Open(s.contentPath(slug, scope))
	if err != nil {
		return nil, fmt.Errorf("open paste: %w", err)
	}
	return f, nil
}

// writeContent writes data atomically to the scope directory and returns bytes written.
func (s *Store) writeContent(scope Scope, slug string, content io.Reader) (int64, error) {
	dir := filepath.Join(s.cfg.DataDir, string(scope))
	tmp, err := os.CreateTemp(dir, ".paste-*")
	if err != nil {
		return 0, fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	n, err := io.Copy(tmp, content)
	if closeErr := tmp.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, fmt.Errorf("write content: %w", err)
	}
	if err := os.Rename(tmpPath, s.contentPath(slug, scope)); err != nil {
		return 0, fmt.Errorf("rename: %w", err)
	}
	return n, nil
}

// writeMetadata writes metadata atomically to the paste's scope.
func (s *Store) writeMetadata(p *models.Paste) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	scope := scopeFor(p.Private)
	tmp, err := os.CreateTemp(s.metaDir(scope), ".meta-*")
	if err != nil {
		return fmt.Errorf("create meta temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write metadata: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close meta temp: %w", err)
	}
	if err := os.Rename(tmpPath, s.metaPath(p.Slug, scope)); err != nil {
		return fmt.Errorf("rename metadata: %w", err)
	}
	return nil
}

// Save writes a new paste atomically.
func (s *Store) Save(p *models.Paste, content io.Reader) error {
	if err := validateSlug(p.Slug); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	scope := scopeFor(p.Private)
	n, err := s.writeContent(scope, p.Slug, content)
	if err != nil {
		return err
	}

	p.Size = n
	return s.writeMetadata(p)
}

// Overwrite replaces an existing paste's content in the same scope.
func (s *Store) Overwrite(slug string, content io.Reader, scope Scope) error {
	if err := validateSlug(slug); err != nil {
		return err
	}

	p, err := s.Get(slug, scope)
	if err != nil {
		return fmt.Errorf("paste not found in %s scope: %w", scope, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	n, err := s.writeContent(scope, slug, content)
	if err != nil {
		return err
	}

	p.Size = n
	return s.writeMetadata(p)
}

// UniqueSlug generates a random slug that does not exist in either scope.
func (s *Store) UniqueSlug() string {
	for {
		slug := utils.Generate()
		if !s.Exists(slug, ScopePublic) && !s.Exists(slug, ScopePrivate) {
			return slug
		}
	}
}
