# naste - Project Context

## What

Minimal paste service. Go stdlib only, zero dependencies. Two binaries:
- `naste-server` - HTTP server on port 8080, stores pastes as flat files
- `naste` - CLI client, pipes text to server, returns URL

## Architecture

```
naste (CLI) → POST /api/paste → naste-server → filesystem
                                  ↓
                          public/ private/ metadata/
```

No database. Atomic writes (temp + rename). 10 MB default size limit.

## Deployment Targets

- Docker (multi-arch: amd64 + arm64 via GitHub Actions)
- Nix flake (packages, overlay, apps)
- NixOS module (systemd system service)
- Home Manager module (systemd user service)
- Bare binary (`go build`)

## Flake Structure

```
flake.nix          - inputs, overlay (builds both binaries), explicit imports
nix/
  app.nix          - perSystem packages (naste, naste-server) + apps
  deploy.nix       - docker deploy app (SSH/local)
  devshell.nix     - devShell + treefmt config (nixfmt + gofmt)
  docker.nix       - buildLayeredImage
  nixos-module.nix - services.naste-server + programs.naste-client options
  home-manager-module.nix - same options, user systemd service
  pre-commit.nix   - git-hooks: treefmt, govet, golangci-lint
```

## Formatting

treefmt-nix with nixfmt + gofmt. Single pre-commit hook (`treefmt`) formats
both nix and go. Run `just fmt` or `nix develop -c treefmt`.

## Secrets

Private paste auth via `PRIVATE_USER` / `PRIVATE_PASS` env vars, or
`PRIVATE_USER_FILE` / `PRIVATE_PASS_FILE` for sops-nix integration.

## Naming

- Repo: `naste` (GitHub: semi710/naste)
- Flake URL: `github:semi710/naste`
- Go module: `github.com/semi710/naste` (pending rename from `nastebin` in go.mod)
- Server binary: `naste-server`
