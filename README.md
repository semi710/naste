# nastebin

A minimal, self-hosted paste service for the command line. No database, no JavaScript, just files.

```bash
$ cat server.log | naste
https://paste.semi.sh/abc123

$ curl https://paste.semi.sh/abc123
<server log contents>
```

## Features

- **Public pastes** — share text via simple URLs
- **Private pastes** — HTTP Basic Auth protected
- **Custom slugs** — readable URLs like `/deploy-script`
- **Zero database** — filesystem-based storage
- **Single binary** — one Go binary, minimal footprint
- **CLI tool** — pipe from anywhere (`naste`)
- **Docker ready** — pre-built images for amd64/arm64
- **Self-host friendly** — works behind Tailscale, no domain needed

---

## Quick Start

### Option 1: Docker (Recommended)

```bash
docker run -d \
  --name naste-server \
  -p 8080:8080 \
  -e PRIVATE_USER=admin \
  -e PRIVATE_PASS=secret \
  -v /var/lib/naste-server/data:/data/paste \
  --restart unless-stopped \
  ghcr.io/semi710/naste-server:latest
```

### Option 2: Pre-built Binary

```bash
# Download latest release
curl -fsSL https://github.com/semi710/nastebin/releases/latest/download/naste-server_linux_amd64 -o naste-server
chmod +x naste-server

# Run
DATA_DIR=./data PORT=8080 ./naste-server
```

### Option 3: Build from Source

```bash
git clone https://github.com/semi710/nastebin
cd nastebin
go build -o naste-server .

DATA_DIR=./data PORT=8080 ./naste-server
```

### Option 4: Nix (for Nix users)

```bash
nix run github:semi710/nastebin -- /path/to/data

# Or deploy with full hardening
nix run github:semi710/nastebin#deploy
nix run github:semi710/nastebin#deploy -- user@server
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `DATA_DIR` | `/data/paste` | Storage directory |
| `PRIVATE_USER` | *(empty)* | Username for private pastes |
| `PRIVATE_PASS` | *(empty)* | Password for private pastes |

**Note:** If `PRIVATE_USER` and `PRIVATE_PASS` are not set, private pastes cannot be created (returns `403 Forbidden`).

### Data Layout

```
DATA_DIR/
├── public/       # Public paste content
├── private/      # Private paste content  
└── metadata/     # JSON metadata per paste
```

---

## CLI Client

### Install

```bash
curl -fsSL https://paste.semi.sh/install | sh
# Installs to /usr/local/bin/naste
```

### Usage

```bash
# Pipe from stdin
echo "hello world" | naste
cat file.txt | naste

# File argument
naste file.txt

# Custom slug
echo "deploy script" | naste --slug deploy

# Private paste
echo "secret" | naste --private

# Force overwrite
echo "updated" | naste --slug deploy --force

# Shorthand flags
echo "quick" | naste -s quick -p
```

### CLI Configuration

Create `~/.config/naste/config.toml`:

```toml
endpoint = "https://paste.example.com"
user = "admin"
password = "secret"
```

Or environment variables:

```bash
export PASTE_ENDPOINT=https://paste.example.com
export PASTE_USER=admin
export PASTE_PASS=secret
```

---

## Production Deployment

### 1. Docker Compose (Simplest)

Create `docker-compose.yml`:

```yaml
services:
  naste-server:
    image: ghcr.io/semi710/naste-server:latest
    container_name: naste-server
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"  # Bind to localhost only
    environment:
      - PORT=8080
      - PRIVATE_USER=${PRIVATE_USER:-}
      - PRIVATE_PASS=${PRIVATE_PASS:-}
    volumes:
      - ./data:/data/paste
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
```

```bash
mkdir -p data
docker compose up -d
```

### 2. With Caddy (HTTPS + Reverse Proxy)

**Why Caddy?** Automatic HTTPS via Let's Encrypt. Single config file.

**Caddyfile:**

```caddy
paste.example.com {
    reverse_proxy localhost:8080
}
```

```bash
# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy

# Configure
sudo cp Caddyfile /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

**With Docker Compose:**

```yaml
services:
  naste-server:
    image: ghcr.io/semi710/naste-server:latest
    container_name: naste-server
    restart: unless-stopped
    expose:
      - "8080"  # Only exposed to other containers, not host
    environment:
      - PORT=8080
      - PRIVATE_USER=${PRIVATE_USER:-}
      - PRIVATE_PASS=${PRIVATE_PASS:-}
    volumes:
      - ./data:/data/paste
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    networks:
      - naste-server

  caddy:
    image: caddy:2-alpine
    container_name: naste-server-caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - naste-server
    depends_on:
      - naste-server

volumes:
  caddy_data:
  caddy_config:

networks:
  naste-server:
    driver: bridge
```

**Caddyfile (for Docker Compose):**

```caddy
paste.example.com {
    reverse_proxy naste-server:8080
}
```

### 3. With Nginx

```nginx
server {
    listen 80;
    server_name paste.example.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

For HTTPS, use certbot:

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d paste.example.com
```

---

## Tailscale Deployment (No Domain Needed)

**Why Tailscale?** Access your paste service from anywhere without opening ports or buying a domain.

### Setup

1. **Install Tailscale** on server:
   ```bash
   curl -fsSL https://tailscale.com/install.sh | sh
   sudo tailscale up
   ```

2. **Get your Tailscale IP:**
   ```bash
   tailscale ip -4
   # 100.x.y.z
   ```

3. **Deploy naste-server** (any method above, but bind to Tailscale IP):
   ```bash
   # Docker - only accessible via Tailscale
   docker run -d \
     --name naste-server \
     -p "100.x.y.z:8080:8080" \
     -e PRIVATE_USER=admin \
     -e PRIVATE_PASS=secret \
     -v /var/lib/naste-server/data:/data/paste \
     ghcr.io/semi710/naste-server:latest
   ```

4. **Access from any device** on your tailnet:
   ```bash
   echo "hello from laptop" | PASTE_ENDPOINT=http://100.x.y.z:8080 naste
   # → http://100.x.y.z:8080/abc123
   ```

**Benefits:**
- No domain registration
- No port forwarding
- Encrypted WireGuard tunnel
- Access control via ACLs
- Works behind CGNAT

### Tailscale + HTTPS (Funnel)

Expose to the public internet securely:

```bash
# On server
sudo tailscale funnel 8080
# → https://your-machine.tailnet-name.ts.net
```

Now anyone can use your paste service:
```bash
echo "shared paste" | PASTE_ENDPOINT=https://your-machine.tailnet-name.ts.net naste
```

---

## Nix Deployment (Advanced)

### Local Build

```bash
nix build github:semi710/nastebin
./result/bin/naste-server
```

### Deploy with Hardening

```bash
# Local
nix run github:semi710/nastebin#deploy

# Remote (SSH)
nix run github:semi710/nastebin#deploy -- user@server

# First run: prompts for PRIVATE_USER, PRIVATE_PASS, PORT
# Saves config to /etc/naste-server/env (persisted across deploys)
```

### As a NixOS Service

```nix
# configuration.nix
services.naste-server = {
  enable = true;
  dataDir = "/var/lib/naste-server";
  port = 8080;
  privateUser = "admin";
  privatePass = config.sops.secrets.naste-server_password.path;  # Use sops-nix
};
```

---

## API Reference

### Create Paste

```bash
# Auto slug
curl -X POST http://localhost:8080/api/paste -d "hello world"
# → {"url":"http://localhost:8080/abc123"}

# Custom slug
curl -X POST http://localhost:8080/api/paste \
  -H "X-Slug: myfile" \
  -d "custom content"

# Private
curl -X POST http://localhost:8080/api/paste \
  -u admin:secret \
  -H "X-Private: true" \
  -d "secret"
```

### Retrieve

```bash
# Public
curl http://localhost:8080/abc123

# Private
curl -u admin:secret http://localhost:8080/private/abc123
```

### Overwrite

```bash
curl -X PUT http://localhost:8080/api/paste/abc123 -d "updated"
```

### Health

```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

---

## Security

- Path traversal prevention (slug validation)
- Atomic file writes (temp + rename)
- Constant-time credential comparison
- Request size limit (10MB)
- HTTP timeouts (prevents slowloris)
- Security headers (CSP, X-Frame-Options, etc.)
- Docker hardening: read-only rootfs, no capabilities, non-root user

### DDoS Protection

**naste-server does not handle DDoS protection at the application layer.** For production deployments, use a reverse proxy or CDN:

**Caddy (recommended):**
```caddy
paste.example.com {
    # Rate limiting via plugin
    # Install: xcaddy build --with github.com/mholt/caddy-ratelimit
    rate_limit {
        zone naste-server {
            key {remote_host}
            events 100
            window 1m
        }
    }
    
    reverse_proxy localhost:8080
}
```

**Nginx:**
```nginx
limit_req_zone $binary_remote_addr zone=paste:10m rate=10r/s;
limit_conn_zone $binary_remote_addr zone=addr:10m;

server {
    location / {
        limit_req zone=paste burst=20 nodelay;
        limit_conn addr 10;
        proxy_pass http://localhost:8080;
    }
}
```

**Why not in the app?**
- Proxies handle millions of requests efficiently (written in C)
- Application-level rate limiting adds latency and mutex contention
- Volumetric attacks (100Gbps) cannot be stopped at the application layer
- For serious protection, use Cloudflare, AWS Shield, or Tailscale Funnel

---

## Development

```bash
# Run tests
go test ./...

# Lint
golangci-lint run ./...

# Build binaries
go build -o naste-server .
go build -o naste ./cmd/naste

# With Nix + just
just test
just lint
just image
just deploy
```

---

## Architecture

```
nastebin/
├── main.go                 # Server entry
├── cmd/naste/main.go       # CLI client
├── internal/
│   ├── config/             # Env-based config
│   ├── handlers/           # HTTP handlers
│   ├── models/             # Paste struct
│   ├── storage/            # Filesystem persistence
│   └── utils/              # Slug generation/validation
└── nix/
    ├── docker.nix          # Docker image builder
    ├── deploy.nix          # Nix deploy app
    └── devshell.nix        # Dev environment
```

---

## License

MIT
