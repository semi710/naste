# naste - Project Memory

> Context for LLMs working on this repo. Read this first.

## What is naste?

A minimal, self-hosted paste service for the command line. Go stdlib only, zero dependencies. Filesystem storage, no database.

## Repo

- GitHub: `github.com/semi710/naste`
- Go module: `github.com/semi710/naste` (currently `github.com/semi710/nastebin` in go.mod, needs rename)
- Docs: `naste.semi.sh`
- Paste server: `paste.semi.sh`

## Binaries

- `naste-server` — HTTP server (port 8080, filesystem storage)
- `naste` — CLI client (pipes text to server, returns URL)

## Architecture

```
naste (CLI) → POST /api/paste → naste-server → filesystem (/data/paste/)
```

Storage layout: `public/`, `private/`, `metadata/` directories. Atomic writes (temp + rename).

## Key Files

| File | Purpose |
|------|---------|
| `main.go` | Server entry point, graceful shutdown |
| `cmd/naste/main.go` | CLI client, language detection, config loading |
| `internal/config/config.go` | Env var config (PORT, DATA_DIR, PRIVATE_*, MAX_PASTE_SIZE, *_FILE) |
| `internal/handlers/handlers.go` | HTTP handlers, syntax highlighting, Basic Auth |
| `internal/handlers/static.go` | HTML templates (landing page, paste view), favicon |
| `internal/handlers/middleware.go` | Security headers |
| `internal/storage/storage.go` | Filesystem persistence, atomic writes |
| `internal/utils/slug.go` | Slug generation + validation |
| `internal/models/paste.go` | Paste struct |

## Nix Files

| File | Purpose |
|------|---------|
| `flake.nix` | Flake definition, overlay, auto-imports `./nix/` |
| `nix/app.nix` | Packages: default (server), naste-server, naste, apps.naste |
| `nix/docker.nix` | Docker image via nix |
| `nix/deploy.nix` | SSH/local deploy script |
| `nix/devshell.nix` | Dev shell (go, gopls, golangci-lint, air, just) |
| `nix/pre-commit.nix` | Git hooks |
| `nix/nixos-module.nix` | NixOS module (services.naste-server + programs.naste-client) |
| `nix/home-manager-module.nix` | Home Manager module (services.naste-server + programs.naste-client) |

## CI/CD

| Workflow | File | Trigger |
|----------|------|---------|
| CI (test, lint, security) | `.github/workflows/ci.yml` | push to master |
| Docker build + push | `.github/workflows/nix-build-push.yml` | push to master |
| Docs build + deploy | `.github/workflows/docs.yml` | push to master (docs/** or mkdocs.yml) |
| Release binaries | `.github/workflows/release.yml` | tag `v*` |

## Releasing

```bash
git tag v0.2.0
git push origin v0.2.0
```

Release workflow builds for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64. Version injected via ldflags (`-X main.version`).

## Environment Variables

### Server

| Var | Default | Description |
|-----|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DATA_DIR` | `/data/paste` | Storage directory |
| `MAX_PASTE_SIZE` | `10485760` | Max paste size in bytes (10 MB) |
| `PRIVATE_USER` | (empty) | Auth username |
| `PRIVATE_PASS` | (empty) | Auth password |
| `PRIVATE_USER_FILE` | (empty) | File containing username (overrides inline) |
| `PRIVATE_PASS_FILE` | (empty) | File containing password (overrides inline) |

### CLI

| Var | Default | Description |
|-----|---------|-------------|
| `PASTE_ENDPOINT` | `https://paste.semi.sh` | Server URL |
| `PASTE_USER` | (empty) | Auth username |
| `PASTE_PASS` | (empty) | Auth password |
| `PASTE_USER_FILE` | (empty) | File containing username |
| `PASTE_PASS_FILE` | (empty) | File containing password |

## NixOS Module Options

### services.naste-server

- `enable` — run as systemd service
- `port` — listen port (8080)
- `dataDir` — storage path (/var/lib/naste-server/data)
- `maxPasteSize` — max paste size in bytes (10 MB)
- `privateUser` / `privatePass` — inline credentials
- `privateUserFile` / `privatePassFile` — file-based credentials (sops-nix)
- `openFirewall` — open firewall port

### programs.naste-client

- `enable` — install naste CLI
- `endpoint` — **required** server URL
- `privateUser` / `privatePass` — inline credentials
- `privateUserFile` / `privatePassFile` — file-based credentials

## Home Manager Module Options

Same as NixOS but:
- Runs as user systemd service (no root)
- No `openFirewall` option
- `dataDir` defaults to `~/.local/share/naste-server/data`
- `programs.naste-client` sets env vars via `home.sessionVariables`

## Deployment

- **obox** (Oracle VPS): runs `services.naste-server` via NixOS module, Caddy reverse proxy at `paste.semi.sh`
- **semi, dsd, mach** (workstations): `programs.naste-client` with private creds via sops
- **nikhil** (home user): `programs.naste-client` without private creds (public only)

## Conventions

- Go 1.25, zero external dependencies
- Ponytail mode: minimal, stdlib first, no abstractions
- Tests for non-trivial logic (config, storage, handlers, utils)
- Pre-commit: gofmt, golangci-lint, govet
- Docs: mkdocs-material, served at naste.semi.sh
- Comments explain why, not what
- No em-dashes
