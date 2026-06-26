# CLI Usage

The `naste` CLI client sends text to a paste server and returns a URL. It can also fetch pastes with the `get` subcommand.

## Installation

=== "Go"

    ```bash
    go install github.com/semi710/naste/cmd/naste@latest
    ```

=== "Nix (run without install)"

    ```bash
    echo "hello" | nix run github:semi710/naste#naste --
    ```

=== "Nix (install)"

    ```bash
    nix profile install github:semi710/naste#naste
    ```

=== "NixOS module"

    ```nix
    programs.naste-client = {
      enable = true;
      endpoint = "https://paste.semi.sh";
    };
    ```

=== "Home Manager"

    Add `naste` to `home.packages` and set `PASTE_ENDPOINT` via sessionVariables.

## Fetching Pastes

Use the `get` subcommand to fetch paste content. You can also use `curl`, but `naste get` handles auth and endpoints automatically:

```bash
# Public paste
naste get hello
# Equivalent: curl https://paste.semi.sh/hello

# Private paste (uses creds from config/env, prompts if missing)
naste get -p hello
# Equivalent: curl -u admin:secret https://paste.semi.sh/private/hello
```

`naste get -p` reads credentials from `PASTE_USER`/`PASTE_PASS` env vars (or `PASTE_USER_FILE`/`PASTE_PASS_FILE` for sops-nix). If no credentials are configured, it prompts for username and password interactively. No need to remember the `/private/` URL prefix or pass `-u` flags manually.

## Flags

| Flag | Shorthand | Description |
|------|-----------|-------------|
| `--slug <name>` | `-s` | Custom slug for the paste |
| `--private` | `-p` | Create/fetch a private paste (requires auth) |
| `--force` | `-f` | Force overwrite if slug exists |
| `--lang <lang>` | `-l` | Language for syntax highlighting (auto-detected from file extension if not set) |
| `--version` | `-v` | Print version and exit |
| `--help` | `-h` | Show help |

## Input Methods

### Pipe from stdin

```bash
echo "hello world" | naste
cat file.go | naste
kubectl get pods | naste -s k8s-pods
```

### File argument

Pass a file path as a positional argument. Language is auto-detected from the extension.

```bash
naste main.go
naste deploy.sh
naste config.yaml
```

## Configuration

### Config file

Create `~/.config/naste/config.toml`:

```toml
endpoint = "https://paste.semi.sh"
user = "admin"
password = "secret"
```

### Environment variables

Environment variables override the config file:

| Variable | Description |
|----------|-------------|
| `PASTE_ENDPOINT` | Server URL (default: `https://paste.semi.sh`) |
| `PASTE_USER` | Username for private pastes |
| `PASTE_PASS` | Password for private pastes |
| `PASTE_USER_FILE` | File containing username (overrides `PASTE_USER`) |
| `PASTE_PASS_FILE` | File containing password (overrides `PASTE_PASS`) |

File-based credentials take precedence over inline env vars. File contents are trimmed of whitespace. Use with sops-nix for secret management.

## Language Detection

Auto-detected from file extension when using file arguments. Override with `--lang`.

| Extension | Language |
|-----------|----------|
| `.go` | go |
| `.py` `.pyw` | python |
| `.js` `.jsx` `.mjs` | javascript |
| `.ts` `.tsx` | typescript |
| `.rs` | rust |
| `.c` `.h` | c |
| `.cpp` `.cxx` `.cc` `.hpp` | cpp |
| `.sh` `.bash` | bash |
| `.nix` | nix |
| `.yaml` `.yml` | yaml |
| ... | 30+ languages total |

## Overwrite Behavior

If a slug already exists in the same scope (public or private) and `--force` is not passed, the CLI prompts interactively:

```bash
$ naste file.go -s mycode
Slug 'mycode' exists. Override? [y/N]
```

The same slug can exist independently in both public and private scopes. The prompt reads from `/dev/tty` (not stdin), so it works even when piping content.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (network, file read, server rejection) |

## Examples

```bash
# Quick paste
echo "hello" | naste

# Named paste with language
cat script.sh | naste -s deploy -l bash

# Private paste
cat secrets.env | naste -p -s secrets

# Overwrite existing
naste updated.go -f -s mycode

# Pipe kubectl output
kubectl get pods -o yaml | naste -s k8s-debug

# Fetch a public paste
naste get hello

# Fetch a private paste
naste get -p hello
```
