{ inputs, lib, ... }:
{
  imports = [
    (inputs.git-hooks + /flake-module.nix)
  ];

  perSystem =
    { config, ... }:
    {
      pre-commit.settings = {
        hooks = {
          govet.enable = true;
          golangci-lint.enable = true;
          treefmt = {
            enable = true;
            entry = "${lib.getExe config.treefmt.build.wrapper}";
          };
        };
      };
    };
}
