package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const version = "0.1.0"

// Config holds CLI configuration.
type Config struct {
	Endpoint string
	User     string
	Password string
}

func main() {
	// Subcommand: naste get [-p] <slug>
	if len(os.Args) > 1 && os.Args[1] == "get" {
		runGet(os.Args[2:])
		return
	}

	var (
		slugOpt    string
		privateOpt bool
		forceOpt   bool
		versionOpt bool
		langOpt    string
	)

	// Define flags
	flag.StringVar(&slugOpt, "slug", "", "Custom slug for the paste")
	flag.StringVar(&slugOpt, "s", "", "(shorthand)")
	flag.BoolVar(&privateOpt, "private", false, "Create a private paste")
	flag.BoolVar(&privateOpt, "p", false, "(shorthand)")
	flag.BoolVar(&forceOpt, "force", false, "Force overwrite if slug exists")
	flag.BoolVar(&forceOpt, "f", false, "(shorthand)")
	flag.StringVar(&langOpt, "lang", "", "Language for syntax highlighting (auto-detected from file extension if not set)")
	flag.StringVar(&langOpt, "l", "", "(shorthand)")
	flag.BoolVar(&versionOpt, "version", false, "Print version and exit")
	flag.BoolVar(&versionOpt, "v", false, "(shorthand)")

	// Custom usage to group shorthand flags
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: naste [OPTIONS] [FILE]\n")
		fmt.Fprintf(os.Stderr, "       naste get [-p] <slug>\n\n")
		fmt.Fprintf(os.Stderr, "Pipe text to create a paste, or pass a file path.\n")
		fmt.Fprintf(os.Stderr, "Use 'naste get' to fetch a paste.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  echo \"hello\" | naste\n")
		fmt.Fprintf(os.Stderr, "  naste file.go\n")
		fmt.Fprintf(os.Stderr, "  echo \"secret\" | naste -p -s mycode\n")
		fmt.Fprintf(os.Stderr, "  cat deploy.sh | naste -l bash\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -s, --slug string     Custom slug for the paste\n")
		fmt.Fprintf(os.Stderr, "  -p, --private         Create a private paste\n")
		fmt.Fprintf(os.Stderr, "  -f, --force           Force overwrite if slug exists\n")
		fmt.Fprintf(os.Stderr, "  -l, --lang string     Language for syntax highlighting\n")
		fmt.Fprintf(os.Stderr, "                          (auto-detected from file extension if not set)\n")
		fmt.Fprintf(os.Stderr, "  -v, --version         Print version and exit\n")
		fmt.Fprintf(os.Stderr, "  -h, --help            Show this help message\n")
	}

	flag.Parse()

	if versionOpt {
		fmt.Println("naste", version)
		os.Exit(0)
	}

	cfg := loadConfig()

	var content []byte
	var err error
	filename := ""

	// Read from file if positional arg provided, else stdin
	if len(flag.Args()) > 0 {
		filename = flag.Args()[0]
		content, err = os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read file: %v\n", err)
			os.Exit(1)
		}
	} else {
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read stdin: %v\n", err)
			os.Exit(1)
		}
	}

	if len(content) == 0 {
		fmt.Fprintln(os.Stderr, "no input provided")
		os.Exit(1)
	}

	slug := slugOpt
	private := privateOpt
	force := forceOpt

	// Auto-detect language from filename if not set
	if langOpt == "" && filename != "" {
		langOpt = detectLang(filename)
	}

	// Configure endpoint: env var > config file > default public server
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://paste.semi.sh"
	}

	// First request: POST (or PUT if forcing overwrite)
	url, method := buildRequest(endpoint, slug, force)
	resp, body := doRequest(cfg, method, url, content, slug, private, force, langOpt)
	defer func() { _ = resp.Body.Close() }()

	// Handle slug conflict with interactive prompt
	if resp.StatusCode == http.StatusConflict && slug != "" && !force {
		if promptOverwrite(slug) {
			url = fmt.Sprintf("%s/api/paste/%s", endpoint, slug)
			resp, body = doRequest(cfg, http.MethodPut, url, content, slug, private, true, langOpt)
			defer func() { _ = resp.Body.Close() }()

			// If PUT returns 404, the slug exists in the other scope, not this one.
			// Fall back to POST to create a new paste in this scope.
			if resp.StatusCode == http.StatusNotFound {
				url, method = buildRequest(endpoint, slug, false)
				resp, body = doRequest(cfg, method, url, content, slug, private, false, langOpt)
				defer func() { _ = resp.Body.Close() }()
			}
		} else {
			os.Exit(1)
		}
	}

	// Check success
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "error: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintf(os.Stderr, "parse response: %v\n", err)
		os.Exit(1)
	}

	if u, ok := result["url"]; ok {
		fmt.Println(u)
	} else if status, ok := result["status"]; ok && status == "updated" {
		fmt.Printf("%s/%s\n", endpoint, slug)
	} else {
		fmt.Println(string(body))
	}
}

var langByExt = map[string]string{
	".go": "go", ".py": "python", ".pyw": "python",
	".js": "javascript", ".jsx": "javascript", ".mjs": "javascript",
	".ts": "typescript", ".tsx": "typescript", ".rs": "rust",
	".c": "c", ".h": "c", ".cpp": "cpp", ".cxx": "cpp", ".cc": "cpp", ".hpp": "cpp",
	".java": "java", ".rb": "ruby", ".sh": "bash", ".bash": "bash",
	".php": "php", ".swift": "swift", ".kt": "kotlin", ".kts": "kotlin",
	".scala": "scala", ".r": "r", ".lua": "lua", ".dart": "dart",
	".elm": "elm", ".hs": "haskell", ".ex": "elixir", ".exs": "elixir",
	".clj": "clojure", ".cljs": "clojure", ".nim": "nim", ".zig": "zig",
	".v": "v", ".vsh": "v", ".fish": "fish", ".ps1": "powershell",
	".yaml": "yaml", ".yml": "yaml", ".json": "json", ".toml": "toml",
	".xml": "xml", ".sql": "sql", ".html": "html", ".htm": "html",
	".css": "css", ".scss": "css", ".sass": "css", ".less": "css",
	".md": "markdown", ".markdown": "markdown", ".dockerfile": "dockerfile",
	".makefile": "makefile", ".mk": "makefile", ".cmake": "cmake", ".nix": "nix",
}

// detectLang returns language from file extension.
func detectLang(filename string) string {
	return langByExt[strings.ToLower(filepath.Ext(filename))]
}

func buildRequest(endpoint, slug string, force bool) (string, string) {
	if slug != "" && force {
		return fmt.Sprintf("%s/api/paste/%s", endpoint, slug), http.MethodPut
	}
	return fmt.Sprintf("%s/api/paste", endpoint), http.MethodPost
}

func doRequest(cfg Config, method, url string, content []byte, slug string, private, force bool, lang string) (*http.Response, []byte) {
	req, err := http.NewRequest(method, url, bytes.NewReader(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}

	if slug != "" && !force {
		req.Header.Set("X-Slug", slug)
	}
	if private {
		req.Header.Set("X-Private", "true")
	}
	if lang != "" {
		req.Header.Set("X-Lang", lang)
	}
	if cfg.User != "" && cfg.Password != "" {
		req.SetBasicAuth(cfg.User, cfg.Password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request: %v\n", err)
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		fmt.Fprintf(os.Stderr, "read response: %v\n", err)
		os.Exit(1)
	}

	return resp, body
}

func promptOverwrite(slug string) bool {
	fmt.Fprintf(os.Stderr, "Slug '%s' exists. Override? [y/N] ", slug)

	// Open controlling terminal to read user input even when stdin is piped
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}
	defer func() { _ = tty.Close() }()

	reader := bufio.NewReader(tty)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	return strings.ToLower(strings.TrimSpace(answer)) == "y"
}

// loadConfig reads config from ~/.config/naste/config.toml or env vars.
func loadConfig() Config {
	var cfg Config

	home, err := os.UserHomeDir()
	if err == nil {
		if data, err := os.ReadFile(filepath.Join(home, ".config", "naste", "config.toml")); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) != 2 {
					continue
				}
				key := strings.TrimSpace(parts[0])
				val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				switch key {
				case "endpoint":
					cfg.Endpoint = val
				case "user":
					cfg.User = val
				case "password":
					cfg.Password = val
				}
			}
		}
	}

	if v := os.Getenv("PASTE_ENDPOINT"); v != "" {
		cfg.Endpoint = v
	}
	if v := os.Getenv("PASTE_USER"); v != "" {
		cfg.User = v
	}
	if v := os.Getenv("PASTE_PASS"); v != "" {
		cfg.Password = v
	}
	if f := os.Getenv("PASTE_USER_FILE"); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			cfg.User = strings.TrimSpace(string(b))
		}
	}
	if f := os.Getenv("PASTE_PASS_FILE"); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			cfg.Password = strings.TrimSpace(string(b))
		}
	}

	return cfg
}

// runGet handles: naste get [-p] <slug>
func runGet(args []string) {
	getFlags := flag.NewFlagSet("get", flag.ExitOnError)
	var privateOpt bool
	getFlags.BoolVar(&privateOpt, "p", false, "Get a private paste")
	getFlags.BoolVar(&privateOpt, "private", false, "(shorthand)")
	if err := getFlags.Parse(args); err != nil {
		os.Exit(1)
	}

	if getFlags.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: naste get [-p] <slug>")
		os.Exit(1)
	}

	slug := getFlags.Arg(0)
	cfg := loadConfig()
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://paste.semi.sh"
	}

	// If private and no creds, prompt for them
	if privateOpt && (cfg.User == "" || cfg.Password == "") {
		cfg.User, cfg.Password = promptCreds()
	}

	path := slug
	if privateOpt {
		path = "private/" + slug
	}
	url := fmt.Sprintf("%s/%s", endpoint, path)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}

	if cfg.User != "" && cfg.Password != "" {
		req.SetBasicAuth(cfg.User, cfg.Password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "error: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	fmt.Print(string(body))
}

// promptCreds asks the user for username and password interactively.
func promptCreds() (string, string) {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot open terminal for credential prompt")
		os.Exit(1)
	}
	defer func() { _ = tty.Close() }()

	reader := bufio.NewReader(tty)

	fmt.Fprint(os.Stderr, "Username: ")
	user, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "read username: %v\n", err)
		os.Exit(1)
	}
	user = strings.TrimSpace(user)

	fmt.Fprint(os.Stderr, "Password: ")
	pass, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "read password: %v\n", err)
		os.Exit(1)
	}
	pass = strings.TrimSpace(pass)

	return user, pass
}
