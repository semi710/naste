# naste - Design Decions

## Module `let` bindings: inside `config`, not module top-level

**Decision:** `let cfg = config.services.naste-server` lives inside `config = let ... in`,
not at the module function top level.

**Why:** When a module is wrapped by flake-parts and consumed via nix-wire
(or similar flake-part-based frameworks), top-level `let` bindings force `config`
evaluation before the module system finishes merging options. This causes
infinite recursion. Moving the `let` inside `config` makes it lazy - evaluated
only when `config` is actually merged.

**Applies to:** `nix/nixos-module.nix` and `nix/home-manager-module.nix`.

**Symptom:** `error: infinite recursion encountered` with the message "if you
get an infinite recursion here, you probably reference `config` in `imports`".

---

## `packages.default` is the CLI, not the server

**Decision:** `packages.default = naste` (the CLI client).

**Why:** Users interact with the CLI. `nix run github:semi710/naste` should
give you the tool you use, not the server daemon. The server is
`nix run ...#naste-server` explicitly.

**Previous:** `packages.default` was `naste-server`, with `naste` as a separate
output. The `naste-server` also aliased `default`, which was misleading.

---

## treefmt over bare nixfmt

**Decision:** Use treefmt-nix with a single pre-commit hook for both nix and go.

**Why:** One hook formats everything. Previously gofmt was a separate pre-commit
hook and nixfmt was a bare `formatter` attribute with no hook at all. Now
`just fmt` or `nix develop -c treefmt` handles both languages, and the
pre-commit hook runs the same treefmt wrapper.

**Standard:** Matches the ndots (nixos-config) repo's formatting setup.

---

## Explicit flake imports over readDir auto-import

**Decision:** `imports = [ ./nix/app.nix ./nix/deploy.nix ... ]` explicitly.

**Why:** The previous `map (fn: ./nix/${fn}) (filter ... (readDir ./nix))`
was clever but fragile. Adding a non-module `.nix` file to `nix/` would silently
break evaluation. Explicit imports are clear and survive restructuring.

---

## Overlay builds, app.nix references

**Decision:** The overlay (`flake.overlays.default`) defines `buildGoModule`
calls. `nix/app.nix` defines the same packages for `perSystem` output.

**Why:** flake-parts `perSystem` pkgs and the overlay are different evaluation
contexts. Sharing via `self'` causes infinite recursion. Both are needed:
overlay for external consumers (`nixpkgs.overlays = [ naste.overlays.default ]`),
perSystem for `nix build .#naste`. The `src` + `commonArgs` let bindings are
duplicated between the two, but this is the smallest working diff.

**Upgrade path:** If flake-parts supports overlay-aware perSystem pkgs without
circular refs, consolidate into one build definition.
