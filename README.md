<div align="center">

# naste

A minimal, self-hosted paste service for the command line.

No database. No frameworks. Just files.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![CI](https://img.shields.io/github/actions/workflow/status/semi710/naste/ci.yml?logo=github&label=CI)](https://github.com/semi710/naste/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/docker-ghcr.io-2496ED?logo=docker&logoColor=white)](https://github.com/semi710/naste/pkgs/container/naste-server)
[![Docs](https://img.shields.io/badge/docs-naste.semi.sh-ff6e40?logo=materialformkdocs&logoColor=white)](https://naste.semi.sh)

</div>

---

```bash
$ echo "hello world" | naste
https://paste.semi.sh/abc123

$ curl https://paste.semi.sh/abc123
hello world
```

## Install

```bash
go install github.com/semi710/naste/cmd/naste@latest
```

Or run without installing:

```bash
echo "hello" | nix run github:semi710/naste#naste --
```

## Deploy

```bash
docker run -d --name naste -p 8080:8080 -v /var/lib/naste/data:/data/paste \
  ghcr.io/semi710/naste-server:latest
```

NixOS:

```nix
services.naste-server.enable = true;
programs.naste-client = {
  enable = true;
  endpoint = "https://paste.semi.sh";
};
```

## Documentation

Full docs at [naste.semi.sh](https://naste.semi.sh) - CLI usage, Docker, Nix, NixOS module, Home Manager, API reference, reverse proxy guides.
