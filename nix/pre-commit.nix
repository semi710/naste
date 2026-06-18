{ inputs, ... }:
{
  imports = [
    (inputs.git-hooks + /flake-module.nix)
  ];

  perSystem =
    { config, pkgs, ... }:
    {
      pre-commit.settings = {
        hooks = {
          gofmt.enable = true;
          govet.enable = true;
          golangci-lint.enable = true;
        };
      };
    };
}
