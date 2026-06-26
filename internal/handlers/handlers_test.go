package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/semi710/nastebin/internal/config"
	"github.com/semi710/nastebin/internal/storage"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir:      dir,
		MaxPasteSize: 10 * 1024 * 1024,
	}
	store, err := storage.NewStore(cfg)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return NewHandler(cfg, store)
}

func TestHealthCheck(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("body = %v", body)
	}
}

func TestCreatePaste(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("POST", "/api/paste", strings.NewReader("hello world"))
	req.Host = "paste.example.com"
	w := httptest.NewRecorder()
	h.CreatePaste(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(body["url"], "http://paste.example.com/") {
		t.Errorf("url = %q", body["url"])
	}
}

func TestCreatePasteWithCustomSlug(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("POST", "/api/paste", strings.NewReader("test"))
	req.Header.Set("X-Slug", "mycode")
	req.Host = "p.test"
	w := httptest.NewRecorder()
	h.CreatePaste(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !h.store.Exists("mycode", storage.ScopePublic) {
		t.Error("paste with custom slug not saved")
	}
}

func TestCreatePasteConflict(t *testing.T) {
	h := newTestHandler(t)
	// First create
	req1 := httptest.NewRequest("POST", "/api/paste", strings.NewReader("v1"))
	req1.Header.Set("X-Slug", "dup")
	h.CreatePaste(httptest.NewRecorder(), req1)

	// Second create with same slug
	req2 := httptest.NewRequest("POST", "/api/paste", strings.NewReader("v2"))
	req2.Header.Set("X-Slug", "dup")
	w := httptest.NewRecorder()
	h.CreatePaste(w, req2)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestCreatePrivatePasteNoAuth(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("POST", "/api/paste", strings.NewReader("secret"))
	req.Header.Set("X-Private", "true")
	w := httptest.NewRecorder()
	h.CreatePaste(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestGetPublicPasteRaw(t *testing.T) {
	h := newTestHandler(t)
	// Create a paste
	createReq := httptest.NewRequest("POST", "/api/paste", strings.NewReader("hello world"))
	createReq.Header.Set("X-Slug", "rawtest")
	h.CreatePaste(httptest.NewRecorder(), createReq)

	// Get raw
	req := httptest.NewRequest("GET", "/rawtest?raw=1", nil)
	req.SetPathValue("slug", "rawtest")
	w := httptest.NewRecorder()
	h.GetPublicPaste(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Body.String() != "hello world" {
		t.Errorf("body = %q, want %q", w.Body.String(), "hello world")
	}
	if w.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("content-type = %q", w.Header().Get("Content-Type"))
	}
}

func TestGetPublicPasteBrowser(t *testing.T) {
	h := newTestHandler(t)
	createReq := httptest.NewRequest("POST", "/api/paste", strings.NewReader("package main"))
	createReq.Header.Set("X-Slug", "htmltest")
	h.CreatePaste(httptest.NewRecorder(), createReq)

	req := httptest.NewRequest("GET", "/htmltest", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.SetPathValue("slug", "htmltest")
	w := httptest.NewRecorder()
	h.GetPublicPaste(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<html") {
		t.Error("expected HTML response for browser")
	}
}

func TestGetPublicPasteNotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	req.SetPathValue("slug", "nonexistent")
	w := httptest.NewRecorder()
	h.GetPublicPaste(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOverwritePaste(t *testing.T) {
	h := newTestHandler(t)
	// Create
	createReq := httptest.NewRequest("POST", "/api/paste", strings.NewReader("old"))
	createReq.Header.Set("X-Slug", "ow")
	h.CreatePaste(httptest.NewRecorder(), createReq)

	// Overwrite
	req := httptest.NewRequest("PUT", "/api/paste/ow", strings.NewReader("new content"))
	req.SetPathValue("slug", "ow")
	w := httptest.NewRecorder()
	h.OverwritePaste(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	// Verify content
	getReq := httptest.NewRequest("GET", "/ow?raw=1", nil)
	getReq.SetPathValue("slug", "ow")
	getW := httptest.NewRecorder()
	h.GetPublicPaste(getW, getReq)

	if getW.Body.String() != "new content" {
		t.Errorf("content = %q, want %q", getW.Body.String(), "new content")
	}
}

func TestLandingPage(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.LandingPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<html") {
		t.Error("expected HTML landing page")
	}
}

func TestPrivatePasteAuth(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir:      dir,
		MaxPasteSize: 10 * 1024 * 1024,
		PrivateUser:  "admin",
		PrivatePass:  "secret",
	}
	store, _ := storage.NewStore(cfg)
	h := NewHandler(cfg, store)

	// Create private paste (requires auth)
	createReq := httptest.NewRequest("POST", "/api/paste", strings.NewReader("private data"))
	createReq.Header.Set("X-Slug", "priv")
	createReq.Header.Set("X-Private", "true")
	createReq.SetBasicAuth("admin", "secret")
	h.CreatePaste(httptest.NewRecorder(), createReq)

	// Get without auth -> 401
	req := httptest.NewRequest("GET", "/private/priv", nil)
	req.SetPathValue("slug", "priv")
	w := httptest.NewRecorder()
	h.AuthMiddleware(h.GetPrivatePaste)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no-auth status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// Get with auth -> 200
	req2 := httptest.NewRequest("GET", "/private/priv", nil)
	req2.SetBasicAuth("admin", "secret")
	req2.SetPathValue("slug", "priv")
	w2 := httptest.NewRecorder()
	h.AuthMiddleware(h.GetPrivatePaste)(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("auth status = %d, want %d", w2.Code, http.StatusOK)
	}
	if w2.Body.String() != "private data" {
		t.Errorf("body = %q", w2.Body.String())
	}
}
