{
  description = "Nastebin paste service";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    systems.url = "github:nix-systems/default";

    git-hooks.url = "github:cachix/git-hooks.nix";
    git-hooks.flake = false;

    treefmt-nix.url = "github:numtide/treefmt-nix";
  };

  outputs =
    inputs:
    let
      overlay =
        final: _prev:
        let
          pkgs = final;
          src = pkgs.lib.cleanSourceWith {
            src = ./.;
            filter =
              path: _type:
              let
                baseName = baseNameOf path;
              in
              !(
                baseName == ".direnv"
                || baseName == ".git"
                || pkgs.lib.hasPrefix "result" baseName
                || baseName == "tmp-data"
                || baseName == "test-data"
              );
          };
          commonArgs = {
            inherit src;
            version = "0.1.0";
            vendorHash = null;
            ldflags = [
              "-s"
              "-w"
            ];
          };
        in
        {
          naste-server = pkgs.buildGoModule (
            commonArgs
            // {
              pname = "naste-server";
              subPackages = [ "." ];
              postInstall = ''
                mv $out/bin/nastebin $out/bin/naste-server
              '';
              meta.mainProgram = "naste-server";
            }
          );
          naste = pkgs.buildGoModule (
            commonArgs
            // {
              pname = "naste";
              subPackages = [ "cmd/naste" ];
              meta.mainProgram = "naste";
            }
          );
        };

      # Wrap a module so package options default to our own packages.
      # Consumers don't need to add the overlay separately.
      withPackages =
        module:
        { pkgs, lib, ... }:
        let
          own = pkgs.extend overlay;
        in
        {
          imports = [ module ];
          services.naste-server.package = lib.mkDefault own.naste-server;
          programs.naste-client.package = lib.mkDefault own.naste;
        };
    in
    inputs.flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      flake.overlays.default = overlay;
      flake.homeModules.default = withPackages ./nix/home-manager-module.nix;
      flake.nixosModules.default = withPackages ./nix/nixos-module.nix;

      imports = [
        inputs.treefmt-nix.flakeModule
        ./nix/app.nix
        ./nix/deploy.nix
        ./nix/devshell.nix
        ./nix/docker.nix
        ./nix/pre-commit.nix
      ];
    };
}
