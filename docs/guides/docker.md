# Docker Deployment

The Docker image is built via Nix for reproducibility and minimal size. Multi-arch manifests (amd64 + arm64) are published to GitHub Container Registry.

## Quick Start

```bash
docker run -d \
  --name naste \
  -p 8080:8080 \
  -v /var/lib/naste/data:/data/paste \
  ghcr.io/semi710/naste-server:latest
```

## With Private Paste Auth

```bash
docker run -d \
  --name naste \
  -p 8080:8080 \
  -e PRIVATE_USER=admin \
  -e PRIVATE_PASS=your-secret-password \
  -v /var/lib/naste/data:/data/paste \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges:true \
  --restart unless-stopped \
  ghcr.io/semi710/naste-server:latest
```

## With Secret Files

For Docker Swarm or Kubernetes, mount secret files instead of passing credentials as env vars:

```bash
docker run -d \
  --name naste \
  -p 8080:8080 \
  -e PRIVATE_USER_FILE=/run/secrets/naste-user \
  -e PRIVATE_PASS_FILE=/run/secrets/naste-pass \
  -v /var/lib/naste/data:/data/paste \
  -v ./secrets/user:/run/secrets/naste-user:ro \
  -v ./secrets/pass:/run/secrets/naste-pass:ro \
  ghcr.io/semi710/naste-server:latest
```

File vars take precedence over inline env vars. File contents are trimmed of whitespace.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DATA_DIR` | `/data/paste` | Storage directory |
| `PRIVATE_USER` | (empty) | Username for private pastes |
| `PRIVATE_PASS` | (empty) | Password for private pastes |
| `PRIVATE_USER_FILE` | (empty) | File containing username (overrides `PRIVATE_USER`) |
| `PRIVATE_PASS_FILE` | (empty) | File containing password (overrides `PRIVATE_PASS`) |
| `MAX_PASTE_SIZE` | `10485760` | Maximum paste size in bytes (default: 10 MB) |

## Docker Compose

```yaml
services:
  naste:
    image: ghcr.io/semi710/naste-server:latest
    ports:
      - "8080:8080"
    volumes:
      - naste-data:/data/paste
    environment:
      PRIVATE_USER: admin
      PRIVATE_PASS: ${NASTE_PASS}
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    restart: unless-stopped

volumes:
  naste-data:
```

## Building the Image Locally

```bash
# Build via Nix
nix build .#dockerImage

# Load into Docker
docker load < result

# The image is tagged as ghcr.io/semi710/naste-server:latest
docker run -p 8080:8080 ghcr.io/semi710/naste-server:latest
```

## Image Details

| | |
|---|---|
| **Base** | Nix built layered image |
| **User** | UID 1000 (non-root) |
| **Working dir** | `/data/paste` |
| **Exposed port** | 8080/tcp |
| **Volume** | `/data/paste` |
| **Architectures** | `linux/amd64`, `linux/arm64` |
