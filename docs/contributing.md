# Contributing

## Development Setup

```bash
# Enter devshell (provides go, gopls, delve, golangci-lint, treefmt)
nix develop

# Or use direnv
direnv allow
```

## Workflow

1. Make changes
2. Format: `just fmt` (runs treefmt for nix + go)
3. Lint: `just lint` (golangci-lint + go vet)
4. Test: `just test`
5. Commit with conventional format: `<action>: <stuff>`

## Commit Style

```
feat: add private paste auth via Basic Auth
fix: resolve infinite recursion in module config
docs: update CLI guide for get subcommand
refactor: replace bool scope params with Scope type
```

Actions: `feat`, `fix`, `docs`, `refactor`, `chore`, `ci`, `test`

## Formatting

treefmt handles both nix and go via a single pre-commit hook. The hook runs automatically on commit. To format manually:

```bash
just fmt
```

## Tests

```bash
just test          # go test ./...
nix develop -c go test ./...   # without just
```

## Nix Module Changes

After changing nix modules, verify they eval:

```bash
nix build .#naste --dry-run
nix eval .#nixosModules.default --apply 'm: builtins.typeOf m'
nix eval .#homeModules.default --apply 'm: builtins.typeOf m'
```

## Docs

Docs are served at [naste.semi.sh](https://naste.semi.sh). To preview locally:

```bash
just doc
```

The docs workflow auto-deploys on push to `master` when `docs/**`, `mkdocs.yml`, or `.github/workflows/docs.yml` change.

## CI

- **CI** (`.github/workflows/ci.yml`): go vet, golangci-lint, govulncheck
- **Docs** (`.github/workflows/docs.yml`): builds and deploys mkdocs to GitHub Pages
- **Release** (`.github/workflows/release.yml`): builds and publishes Docker image on tag push

## Project Structure

```
cmd/naste/           CLI client
internal/
  config/            env var config loading
  handlers/          HTTP handlers + syntax highlighting
  models/            Paste struct
  storage/           filesystem storage (per-scope metadata)
  utils/             slug generation + validation
nix/
  app.nix            perSystem packages + apps
  deploy.nix         docker deploy script
  devshell.nix       devShell + treefmt config
  docker.nix         docker image build
  nixos-module.nix   NixOS system service module
  home-manager-module.nix  Home Manager user service module
  pre-commit.nix     git-hooks config
docs/                mkdocs documentation
```
