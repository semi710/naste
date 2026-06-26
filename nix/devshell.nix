{ ... }:
{
  perSystem =
    {
      config,
      pkgs,
      ...
    }:
    {
      treefmt = {
        projectRootFile = "flake.nix";
        programs.nixfmt.enable = true;
        programs.gofmt.enable = true;
      };

      devShells.default = pkgs.mkShell {
        name = "go-devshell";
        inputsFrom = [ config.pre-commit.devShell ];
        packages = with pkgs; [
          go
          gopls
          delve
          golangci-lint
          air
          just
        ];
        shellHook = ''
          echo 1>&2 "go: $(go version)"
          echo 1>&2 "🧬: $(nix eval --raw --impure --expr 'builtins.currentSystem')"
        '';
      };
    };
}
