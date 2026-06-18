{
  description = "Nastebin paste service";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    systems.url = "github:nix-systems/default";

    # git hooks for pre-commit
    git-hooks.url = "github:cachix/git-hooks.nix";
    git-hooks.flake = false;
  };
  outputs =
    inputs:
    inputs.flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;
      debug = true;

      imports = with builtins; map (fn: ./nix/${fn}) (attrNames (readDir ./nix));
    };
}
