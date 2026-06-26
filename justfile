# naste deployment commands

# Show all available commands
[private]
default:
    @just --list

# Build the Go server binary
build:
    go build -o naste-server .

# Build the CLI client
build-cli:
    go build -o naste ./cmd/naste

# Run locally with default settings
run:
    DATA_DIR=./tmp-data PORT=8080 ./naste-server

# Run with auth configured
run-auth:
    DATA_DIR=./tmp-data PORT=8080 PRIVATE_USER=admin PRIVATE_PASS=secret ./naste-server

# Run tests
test:
    go test ./...

# Lint everything
lint:
    golangci-lint run ./...
    go vet ./...

# Format nix + go files via treefmt
fmt:
    nix develop -c treefmt

# Build Docker image via Nix
image:
    nix build .#dockerImage
    echo "Image: ./result"
    ls -la result

# Load image into Docker (requires Docker daemon)
image-load: image
    docker load < result

# Deploy locally
# First run: prompts for PRIVATE_USER, PRIVATE_PASS, PORT
# Subsequent runs: reads from /etc/naste-server/env (preserves config)
deploy:
    nix run .#deploy

# Deploy to remote server via SSH
# First run: prompts for config on remote server
# Subsequent runs: preserves config automatically
# Usage: just deploy-remote user@server.example.com
deploy-remote host:
    nix run .#deploy -- "{{host}}"

# Deploy from GitHub without cloning
# Usage: just deploy-github user@server.example.com
deploy-github host="":
    nix run github:semi710/naste#deploy -- "{{host}}"

# Clean build artifacts
clean:
    rm -f naste-server naste
    rm -rf result result-* tmp-data test-data

# Serve docs locally (http://0.0.0.0:<random-port>)
doc:
    PORT=$(shuf -i 8000-9000 -n 1) && echo "→ http://0.0.0.0:$PORT" && nix run .#docs -- serve -a 0.0.0.0:$PORT --quiet 2>&1 | grep -v "│"
