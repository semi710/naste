# naste

A minimal, self-hosted paste service for the command line. **No database. No frameworks. Just files.**

Host your own for privacy or provide it as a free service to others.

## At a Glance

| | |
|---|---|
| **Language** | Go (stdlib only, zero dependencies) |
| **Storage** | Filesystem (no database) |
| **Binaries** | `naste-server` (server), `naste` (CLI client) |
| **Deployment** | Docker, Nix, NixOS module, Home Manager module |
| **Security** | Path traversal prevention, atomic writes, constant-time auth |
| **Size limit** | 10 MB per paste |

## Quick Start

### CLI Client

The `naste` binary sends text to a paste server and returns a URL. Install it on any machine to push pastes.

=== "Go"

    ```bash
    go install github.com/semi710/naste/cmd/naste@latest
    echo "hello world" | naste
    ```

=== "Nix (run without install)"

    ```bash
    echo "hello world" | nix run github:semi710/naste#naste --
    ```

=== "NixOS module"

    ```nix
    programs.naste-client = {
      enable = true;
      endpoint = "https://paste.semi.sh";
    };
    ```

=== "Home Manager"

    ```nix
    programs.naste-client = {
      enable = true;
      endpoint = "https://paste.semi.sh";
    };
    ```

### Server

The `naste-server` binary hosts the paste service on port 8080. It stores pastes as files on disk. No database, no dependencies.

=== "Docker"

    ```bash
    docker run -d \
      --name naste \
      -p 8080:8080 \
      -v /var/lib/naste/data:/data/paste \
      ghcr.io/semi710/naste-server:latest
    ```

=== "Nix"

    ```bash
    # Default settings (port 8080, data in /data/paste)
    nix run github:semi710/naste#naste-server

    # With custom env vars
    PORT=9090 DATA_DIR=/tmp/naste-data nix run github:semi710/naste#naste-server
    ```

=== "Binary"

    ```bash
    go build -o naste-server .
    PORT=8080 DATA_DIR=./data ./naste-server
    ```

=== "NixOS module"

    ```nix
    services.naste-server = {
      enable = true;
      port = 8080;
      openFirewall = true;
    };
    ```

=== "Home Manager"

    ```nix
    services.naste-server = {
      enable = true;
      port = 8080;
    };
    ```

## Usage

```bash
# Pipe any text
echo "hello world" | naste
# -> https://paste.semi.sh/abc123

# With a custom slug
cat deploy.sh | naste --slug deploy

# Private paste (requires server auth config)
cat secrets.env | naste --private --slug secrets

# Force overwrite existing slug
naste updated.go --force --slug mycode

# Auto-detects language from file extension
naste main.go
```

## Features

- :fontawesome-solid-database: **Zero Database** - Filesystem-based storage. No PostgreSQL, no MongoDB, no Redis.
- :fontawesome-solid-lock: **Private Pastes** - HTTP Basic Auth for sensitive content. No accounts needed.
- :fontawesome-solid-link: **Custom Slugs** - Readable URLs like `/deploy-script` instead of random IDs.
- :fontawesome-solid-code: **Syntax Highlighting** - Auto-detects language from file extension. Dark theme in browser.
- :fontawesome-solid-box: **Single Binary** - One Go binary. Deploy anywhere in seconds.
- :fontawesome-brands-docker: **Docker Ready** - Multi-arch images (amd64 + arm64) via GitHub Actions.
- :fontawesome-brands-linux: **NixOS Module** - Declarative system service with sops-nix secret support.
- :fontawesome-solid-house: **Home Manager** - Run as a user service, no root required.

## Architecture

```
                    +-------------+
                    |   naste     |  CLI client
                    |  (binary)   |  reads ~/.config/naste/config.toml
                    +-----+-------+
                          |
                    POST /api/paste
                    GET  /{slug}
                          |
                    +-----+-------+
                    | naste-server|  HTTP server
                    |  (binary)   |  port 8080
                    +-----+-------+
                          |
                    +-----+-------+
                    | filesystem  |  /data/paste/
                    |  storage    |  public/ private/ metadata/
                    +-------------+
```

The server stores pastes as flat files in three directories:

- `public/` - public paste content
- `private/` - private paste content (auth-gated)
- `metadata/` - JSON metadata per paste (slug, lang, size, timestamps)

All writes are atomic (temp file + rename). No partial states.

## Configuration

The CLI reads configuration from (in order of precedence):

1. `PASTE_ENDPOINT` env var
2. `PASTE_USER` / `PASTE_PASS` / `PASTE_USER_FILE` / `PASTE_PASS_FILE` env vars
3. `~/.config/naste/config.toml`
4. Built-in default: `https://paste.semi.sh`

```toml
# ~/.config/naste/config.toml
endpoint = "https://paste.semi.sh"
user = "admin"
password = "secret"
```

## Documentation

Docs are served at [naste.semi.sh](https://naste.semi.sh). To preview locally: `just doc`

- [CLI Usage](guides/cli.md) - All flags and options
- [Docker](guides/docker.md) - Container deployment
- [Nix](guides/nix.md) - Nix flake usage
- [NixOS](guides/nixos.md) - System service with sops-nix, all options
- [Home Manager](guides/home-manager.md) - User service, all options
- [Reverse Proxy](guides/reverse-proxy.md) - Caddy, Nginx, Tailscale
- [API Reference](api/index.md) - HTTP endpoints
