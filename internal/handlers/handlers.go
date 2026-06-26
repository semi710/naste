package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/semi710/nastebin/internal/config"
	"github.com/semi710/nastebin/internal/models"
	"github.com/semi710/nastebin/internal/storage"
	"github.com/semi710/nastebin/internal/utils"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	cfg   *config.Config
	store *storage.Store
}

func NewHandler(cfg *config.Config, store *storage.Store) *Handler {
	return &Handler{cfg: cfg, store: store}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.HealthCheck)
	mux.HandleFunc("POST /api/paste", h.CreatePaste)
	mux.HandleFunc("PUT /api/paste/{slug}", h.OverwritePaste)
	mux.HandleFunc("GET /private/{slug}", h.AuthMiddleware(h.GetPrivatePaste))
	mux.HandleFunc("GET /{slug}", h.GetPublicPaste)
	mux.HandleFunc("GET /", h.LandingPage)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// schemeFrom detects the URL scheme from TLS state or X-Forwarded-Proto.
func schemeFrom(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS == nil {
		return "http"
	}
	return "https"
}

// pasteURL builds the public or private URL for a paste.
func pasteURL(r *http.Request, slug string, scope storage.Scope) string {
	scheme := schemeFrom(r)
	if scope == storage.ScopePrivate {
		return fmt.Sprintf("%s://%s/private/%s", scheme, r.Host, slug)
	}
	return fmt.Sprintf("%s://%s/%s", scheme, r.Host, slug)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// scopeFromRequest reads X-Private header and returns the scope.
func scopeFromRequest(r *http.Request) storage.Scope {
	if r.Header.Get("X-Private") == "true" {
		return storage.ScopePrivate
	}
	return storage.ScopePublic
}

// requirePrivateAuth checks server config and client creds for private scope.
// Returns true if the request should continue, false if it was rejected.
func (h *Handler) requirePrivateAuth(w http.ResponseWriter, r *http.Request) bool {
	if h.cfg.PrivateUser == "" || h.cfg.PrivatePass == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "private_auth_not_configured"})
		return false
	}
	if !h.checkBasicAuth(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Private Paste"`)
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func (h *Handler) CreatePaste(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxPasteSize)

	var slug string
	if s := r.Header.Get("X-Slug"); s != "" {
		slug = s
	} else {
		slug = h.store.UniqueSlug()
	}

	if err := utils.Validate(slug); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_slug"})
		return
	}

	scope := scopeFromRequest(r)

	if scope == storage.ScopePrivate && !h.requirePrivateAuth(w, r) {
		return
	}

	if h.store.Exists(slug, scope) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "slug_exists"})
		return
	}

	content, err := io.ReadAll(r.Body)
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		log.Printf("read body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = r.Body.Close()

	paste := &models.Paste{
		Slug:      slug,
		Private:   scope == storage.ScopePrivate,
		Lang:      r.Header.Get("X-Lang"),
		CreatedAt: time.Now().UTC(),
		TTL:       "never",
	}

	if err := h.store.Save(paste, strings.NewReader(string(content))); err != nil {
		log.Printf("save paste: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"url": pasteURL(r, slug, scope)})
}

func (h *Handler) OverwritePaste(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxPasteSize)

	slug := r.PathValue("slug")

	if err := utils.Validate(slug); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	scope := scopeFromRequest(r)

	if scope == storage.ScopePrivate && !h.requirePrivateAuth(w, r) {
		return
	}

	if !h.store.Exists(slug, scope) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "slug_not_found_in_scope"})
		return
	}

	content, err := io.ReadAll(r.Body)
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		log.Printf("read body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = r.Body.Close()

	if err := h.store.Overwrite(slug, strings.NewReader(string(content)), scope); err != nil {
		log.Printf("overwrite paste: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "url": pasteURL(r, slug, scope)})
}

func (h *Handler) GetPublicPaste(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if err := utils.Validate(slug); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	meta, err := h.store.Get(slug, storage.ScopePublic)
	if err != nil || meta.Private {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	content, err := h.store.GetContent(slug, storage.ScopePublic)
	if err != nil {
		log.Printf("get content: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() { _ = content.Close() }()

	data, err := io.ReadAll(content)
	if err != nil {
		log.Printf("read content: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("raw") == "1" || !isBrowser(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	highlighted := highlightSyntax(string(data))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, pasteViewHTML, slug, slug, formatSize(meta.Size), highlighted)
}

func (h *Handler) GetPrivatePaste(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if err := utils.Validate(slug); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	meta, err := h.store.Get(slug, storage.ScopePrivate)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	content, err := h.store.GetContent(slug, storage.ScopePrivate)
	if err != nil {
		log.Printf("get content: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() { _ = content.Close() }()

	data, err := io.ReadAll(content)
	if err != nil {
		log.Printf("read content: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("raw") == "1" || !isBrowser(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	highlighted := highlightSyntax(string(data))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, pasteViewHTML, slug, slug, formatSize(meta.Size), highlighted)
}

func (h *Handler) LandingPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(landingHTML))
}

func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.checkBasicAuth(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Private Paste"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (h *Handler) checkBasicAuth(r *http.Request) bool {
	if h.cfg.PrivateUser == "" || h.cfg.PrivatePass == "" {
		return false
	}
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(h.cfg.PrivateUser)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(h.cfg.PrivatePass)) == 1
	return userMatch && passMatch
}
