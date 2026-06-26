package storage

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/semi710/nastebin/internal/config"
	"github.com/semi710/nastebin/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir}
	s, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestNewStoreCreatesDirs(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir}
	_, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	for _, sub := range []string{"public", "private", "metadata/public", "metadata/private"} {
		if _, err := os.Stat(filepath.Join(dir, sub)); err != nil {
			t.Errorf("dir %s not created: %v", sub, err)
		}
	}
}

func TestSaveAndGet(t *testing.T) {
	s := newTestStore(t)
	p := &models.Paste{Slug: "test", Private: false, Lang: "go"}
	if err := s.Save(p, strings.NewReader("hello world")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !s.Exists("test", ScopePublic) {
		t.Error("Exists returned false after Save")
	}

	meta, err := s.Get("test", ScopePublic)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if meta.Slug != "test" || meta.Size != 11 {
		t.Errorf("meta = %+v", meta)
	}

	rc, err := s.GetContent("test", ScopePublic)
	if err != nil {
		t.Fatalf("GetContent: %v", err)
	}
	defer func() { _ = rc.Close() }()
	data, _ := io.ReadAll(rc)
	if string(data) != "hello world" {
		t.Errorf("content = %q, want %q", data, "hello world")
	}
}

func TestSavePrivate(t *testing.T) {
	s := newTestStore(t)
	p := &models.Paste{Slug: "secret", Private: true}
	if err := s.Save(p, strings.NewReader("private data")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(s.cfg.DataDir, "public", "secret")); !os.IsNotExist(err) {
		t.Error("private paste found in public dir")
	}

	rc, err := s.GetContent("secret", ScopePrivate)
	if err != nil {
		t.Fatalf("GetContent private: %v", err)
	}
	defer func() { _ = rc.Close() }()
	data, _ := io.ReadAll(rc)
	if string(data) != "private data" {
		t.Errorf("content = %q", data)
	}
}

func TestOverwrite(t *testing.T) {
	s := newTestStore(t)
	p := &models.Paste{Slug: "ow", Private: false}
	if err := s.Save(p, strings.NewReader("old")); err != nil {
		t.Fatal(err)
	}

	if err := s.Overwrite("ow", strings.NewReader("new content"), ScopePublic); err != nil {
		t.Fatalf("Overwrite: %v", err)
	}

	rc, _ := s.GetContent("ow", ScopePublic)
	defer func() { _ = rc.Close() }()
	data, _ := io.ReadAll(rc)
	if string(data) != "new content" {
		t.Errorf("content = %q, want 'new content'", data)
	}

	meta, _ := s.Get("ow", ScopePublic)
	if meta.Size != 11 {
		t.Errorf("size = %d, want 11", meta.Size)
	}
}

func TestExistsMissing(t *testing.T) {
	s := newTestStore(t)
	if s.Exists("nonexistent", ScopePublic) {
		t.Error("Exists returned true for missing slug")
	}
}

func TestValidateSlugRejectsTraversal(t *testing.T) {
	if err := validateSlug("../etc"); err == nil {
		t.Error("expected error for path traversal slug")
	}
	if err := validateSlug("foo/bar"); err == nil {
		t.Error("expected error for slash in slug")
	}
	if err := validateSlug("foo\\bar"); err == nil {
		t.Error("expected error for backslash in slug")
	}
}

func TestUniqueSlug(t *testing.T) {
	s := newTestStore(t)
	slug := s.UniqueSlug()
	if slug == "" {
		t.Error("UniqueSlug returned empty")
	}
	if s.Exists(slug, ScopePublic) || s.Exists(slug, ScopePrivate) {
		t.Error("UniqueSlug returned a slug that already exists")
	}
}

func TestSameSlugBothScopes(t *testing.T) {
	s := newTestStore(t)

	pub := &models.Paste{Slug: "shared", Private: false}
	if err := s.Save(pub, strings.NewReader("public content")); err != nil {
		t.Fatal(err)
	}

	priv := &models.Paste{Slug: "shared", Private: true}
	if err := s.Save(priv, strings.NewReader("private content")); err != nil {
		t.Fatal(err)
	}

	if !s.Exists("shared", ScopePublic) {
		t.Error("public scope missing")
	}
	if !s.Exists("shared", ScopePrivate) {
		t.Error("private scope missing")
	}

	pubMeta, _ := s.Get("shared", ScopePublic)
	if pubMeta.Private {
		t.Error("public metadata says private")
	}

	privMeta, _ := s.Get("shared", ScopePrivate)
	if !privMeta.Private {
		t.Error("private metadata says public")
	}

	pubContent, _ := s.GetContent("shared", ScopePublic)
	defer func() { _ = pubContent.Close() }()
	pubData, _ := io.ReadAll(pubContent)
	if string(pubData) != "public content" {
		t.Errorf("public content = %q", pubData)
	}

	privContent, _ := s.GetContent("shared", ScopePrivate)
	defer func() { _ = privContent.Close() }()
	privData, _ := io.ReadAll(privContent)
	if string(privData) != "private content" {
		t.Errorf("private content = %q", privData)
	}
}
