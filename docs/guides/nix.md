# Nix Deployment

The naste flake provides packages, apps, an overlay, and modules for Nix-based deployment.

## Flake Outputs

| Output | Type | Description |
|--------|------|-------------|
| `packages.default` | derivation | `naste` CLI binary |
| `packages.naste` | derivation | `naste` CLI binary (same as default) |
| `packages.naste-server` | derivation | `naste-server` binary |
| `packages.dockerImage` | derivation | Docker image tarball |
| `apps.naste` | app | Run `naste` CLI directly |
| `apps.deploy` | app | SSH/local deploy script |
| `overlays.default` | overlay | Adds `naste-server` and `naste` to pkgs |
| `nixosModules.default` | module | NixOS system service + CLI client |
| `homeModules.default` | module | Home Manager user service |

## Run Without Installing

### Server

```bash
# Default settings (port 8080, data in /data/paste)
nix run github:semi710/naste#naste-server

# With custom env vars
PORT=9090 DATA_DIR=/tmp/naste-data nix run github:semi710/naste#naste-server
```

### CLI

```bash
echo "hello" | nix run github:semi710/naste#naste --
```

## Build

```bash
# Build CLI binary (default)
nix build github:semi710/naste

# Build server binary
nix build github:semi710/naste#naste-server

# Build Docker image
nix build github:semi710/naste#dockerImage
```

## Install

```bash
# Install CLI to profile (default)
nix profile install github:semi710/naste

# Install server to profile
nix profile install github:semi710/naste#naste-server
```

## Deploy Script

The flake includes a deploy app that wraps Docker deployment with config persistence:

```bash
# Deploy locally (prompts for config on first run)
nix run github:semi710/naste#deploy

# Deploy to remote server via SSH
nix run github:semi710/naste#deploy -- user@server

# Deploy from GitHub without cloning
nix run github:semi710/naste#deploy -- user@server.example.com
```

The deploy script:

1. Pulls the latest Docker image from GHCR
2. Reads config from `/etc/naste-server/env` (persists across deploys)
3. Prompts for `PRIVATE_USER`, `PRIVATE_PASS`, `PORT` on first run
4. Stops and removes existing container
5. Starts new container with security hardening (read-only, cap-drop, no-new-privileges)
6. Config is saved to `/etc/naste-server/env` with mode 0600

## Using the Overlay

Add the overlay to your own flake to get `pkgs.naste-server` and `pkgs.naste`:

```nix
{
  inputs.naste.url = "github:semi710/naste";

  outputs = { self, nixpkgs, naste, ... }: {
    nixpkgs.overlays = [ naste.overlays.default ];

    # Now pkgs.naste-server and pkgs.naste are available
  };
}
```

## Justfile Integration

The repo includes a `justfile` with convenient commands:

```bash
just build        # Build server binary
just build-cli    # Build CLI binary
just run          # Run server locally
just run-auth     # Run server with auth
just test         # Run tests
just lint         # golangci-lint + go vet
just fmt          # Format nix + go via treefmt
just image        # Build Docker image via Nix
just image-load   # Build and load into Docker
just deploy       # Deploy locally via Nix
just deploy-remote user@host  # Deploy to remote server
just doc          # Serve docs locally
just clean        # Remove build artifacts
```
