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

// NewHandler creates a Handler with injected dependencies.
func NewHandler(cfg *config.Config, store *storage.Store) *Handler {
	return &Handler{cfg: cfg, store: store}
}

// RegisterRoutes sets up all HTTP routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.HealthCheck)
	mux.HandleFunc("GET /install", h.InstallScript)
	mux.HandleFunc("POST /api/paste", h.CreatePaste)
	mux.HandleFunc("PUT /api/paste/{slug}", h.OverwritePaste)
	mux.HandleFunc("GET /private/{slug}", h.AuthMiddleware(h.GetPrivatePaste))
	mux.HandleFunc("GET /{slug}", h.GetPublicPaste)
	mux.HandleFunc("GET /", h.LandingPage)
}

// HealthCheck responds with a simple status JSON.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// InstallScript serves the CLI install shell script.
func (h *Handler) InstallScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	script := `#!/bin/sh
# naste CLI installer
set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

URL="https://github.com/semi710/nastebin/releases/latest/download/naste_${OS}_${ARCH}"

echo "Downloading naste CLI..."
curl -fsSL "$URL" -o /tmp/naste
chmod +x /tmp/naste

echo "Installing to /usr/local/bin/naste..."
sudo mv /tmp/naste /usr/local/bin/naste

echo "Installed successfully! Run 'naste --help' to get started."
`
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(script))
}

// CreatePaste handles POST /api/paste.
func (h *Handler) CreatePaste(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxPasteSize)

	var slug string
	if s := r.Header.Get("X-Slug"); s != "" {
		slug = s
	} else {
		slug = h.store.UniqueSlug()
	}

	if err := utils.Validate(slug); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_slug"})
		return
	}

	private := r.Header.Get("X-Private") == "true"

	// Reject private paste creation if auth is not configured
	if private && (h.cfg.PrivateUser == "" || h.cfg.PrivatePass == "") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "private_auth_not_configured"})
		return
	}

	if h.store.Exists(slug) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "slug_exists"})
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
		Private:   private,
		Lang:      r.Header.Get("X-Lang"),
		CreatedAt: time.Now().UTC(),
		TTL:       "never",
	}

	if err := h.store.Save(paste, strings.NewReader(string(content))); err != nil {
		log.Printf("save paste: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	host := r.Host

	var url string
	if private {
		url = fmt.Sprintf("%s://%s/private/%s", scheme, host, slug)
	} else {
		url = fmt.Sprintf("%s://%s/%s", scheme, host, slug)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// OverwritePaste handles PUT /api/paste/{slug}.
func (h *Handler) OverwritePaste(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxPasteSize)

	slug := r.PathValue("slug")

	if err := utils.Validate(slug); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Check auth for private pastes
	meta, err := h.store.Get(slug)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if meta.Private {
		if !h.checkBasicAuth(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Private Paste"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
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

	if err := h.store.Overwrite(slug, strings.NewReader(string(content))); err != nil {
		log.Printf("overwrite paste: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// GetPublicPaste serves paste content (HTML for browsers, raw text for curl/tools).
func (h *Handler) GetPublicPaste(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if err := utils.Validate(slug); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	meta, err := h.store.Get(slug)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if meta.Private {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	content, err := h.store.GetContent(slug, false)
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

	// Serve raw text if requested or if not a browser
	if r.URL.Query().Get("raw") == "1" || !isBrowser(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	// Serve HTML with syntax highlighting for browsers
	highlighted := highlightSyntax(string(data))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Override CSP to allow inline styles for syntax highlighting
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, pasteViewHTML, slug, slug, formatSize(meta.Size), highlighted)
}

// formatSize returns human-readable size string.
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

// highlightSyntax adds basic syntax highlighting to code.
// Supports: keywords, strings, comments, numbers, functions.
func highlightSyntax(code string) string {
	keywords := map[string]bool{
		"func": true, "var": true, "const": true, "type": true, "struct": true,
		"interface": true, "package": true, "import": true, "return": true,
		"if": true, "else": true, "for": true, "range": true, "switch": true,
		"case": true, "default": true, "break": true, "continue": true,
		"map": true, "chan": true, "go": true, "defer": true, "select": true,
		"true": true, "false": true, "nil": true, "int": true, "string": true,
		"bool": true, "float64": true, "error": true, "make": true, "new": true,
		"append": true, "len": true, "cap": true, "copy": true, "delete": true,
		"fmt": true, "log": true, "http": true, "os": true, "io": true,
	}

	var result []byte
	i := 0
	for i < len(code) {
		// Comments: // or /* */
		if i+1 < len(code) && code[i] == '/' && code[i+1] == '/' {
			end := i + 2
			for end < len(code) && code[end] != '\n' {
				end++
			}
			result = append(result, `<span class="comment">`...)
			for j := i; j < end; j++ {
				switch code[j] {
				case '<':
					result = append(result, "&lt;"...)
				case '>':
					result = append(result, "&gt;"...)
				case '&':
					result = append(result, "&amp;"...)
				default:
					result = append(result, code[j])
				}
			}
			result = append(result, "</span>"...)
			i = end
			continue
		}

		// Strings: "..." or `...`
		if code[i] == '"' || code[i] == '`' {
			quote := code[i]
			end := i + 1
			for end < len(code) && code[end] != quote {
				if code[end] == '\\' && end+1 < len(code) {
					end += 2
				} else {
					end++
				}
			}
			if end < len(code) {
				end++ // include closing quote
			}
			result = append(result, `<span class="string">`...)
			for j := i; j < end; j++ {
				switch code[j] {
				case '<':
					result = append(result, "&lt;"...)
				case '>':
					result = append(result, "&gt;"...)
				case '&':
					result = append(result, "&amp;"...)
				default:
					result = append(result, code[j])
				}
			}
			result = append(result, "</span>"...)
			i = end
			continue
		}

		// Numbers
		if isDigit(code[i]) {
			start := i
			for i < len(code) && (isDigit(code[i]) || code[i] == '.' || code[i] == '_') {
				i++
			}
			result = append(result, `<span class="number">`...)
			for j := start; j < i; j++ {
				result = append(result, code[j])
			}
			result = append(result, "</span>"...)
			continue
		}

		// Words (keywords, functions)
		if isWordChar(code[i]) {
			start := i
			for i < len(code) && isWordChar(code[i]) {
				i++
			}
			word := code[start:i]
			if keywords[word] {
				result = append(result, fmt.Sprintf(`<span class="keyword">%s</span>`, escapeHTML(word))...)
			} else if i < len(code) && code[i] == '(' {
				result = append(result, fmt.Sprintf(`<span class="function">%s</span>`, escapeHTML(word))...)
			} else {
				result = append(result, escapeHTML(word)...)
			}
			continue
		}

		// Operators
		if isOperator(code[i]) {
			result = append(result, `<span class="operator">`...)
			result = append(result, escapeHTML(string(code[i]))...)
			result = append(result, "</span>"...)
			i++
			continue
		}

		// Regular character (escape HTML)
		switch code[i] {
		case '<':
			result = append(result, "&lt;"...)
		case '>':
			result = append(result, "&gt;"...)
		case '&':
			result = append(result, "&amp;"...)
		case '\t':
			result = append(result, "    "...) // 4 spaces for tabs
		default:
			result = append(result, code[i])
		}
		i++
	}

	return string(result)
}

func isDigit(c byte) bool    { return c >= '0' && c <= '9' }
func isWordChar(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' }
func isOperator(c byte) bool {
	return c == '+' || c == '-' || c == '*' || c == '/' || c == '=' || c == '<' || c == '>' ||
		c == '!' || c == '&' || c == '|' || c == '^' || c == '~' || c == '%'
}
func escapeHTML(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			result = append(result, "&lt;"...)
		case '>':
			result = append(result, "&gt;"...)
		case '&':
			result = append(result, "&amp;"...)
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}

// GetPrivatePaste serves raw text for private pastes (requires auth).
func (h *Handler) GetPrivatePaste(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if err := utils.Validate(slug); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	meta, err := h.store.Get(slug)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !meta.Private {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	content, err := h.store.GetContent(slug, true)
	if err != nil {
		log.Printf("get content: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() { _ = content.Close() }()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, content)
}

// LandingPage serves the HTML landing page.
func (h *Handler) LandingPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(landingHTML))
}

// AuthMiddleware wraps a handler with basic auth for private routes.
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

// checkBasicAuth verifies credentials against configured values.
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
