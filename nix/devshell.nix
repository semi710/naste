{ ... }:
{
  perSystem =
    {
      config,
      pkgs,
      self',
      ...
    }:
    {
      formatter = pkgs.nixfmt;

      devShells.default = pkgs.mkShell {
        name = "go-devshell";
        inputsFrom = [ config.pre-commit.devShell ];
        packages = with pkgs; [
          go
          gopls
          delve
          golangci-lint
        ];
        shellHook = ''
          echo 1>&2 "go: $(go version)"
          echo 1>&2 "🧬: $(nix eval --raw --impure --expr 'builtins.currentSystem')"
        '';
      };
    };
}
