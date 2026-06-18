{ lib, pkgs, ... }:
let
  src = lib.cleanSourceWith {
    src = ../.;
    filter = path: type:
      let
        baseName = baseNameOf path;
      in
      !(baseName == ".direnv"
        || baseName == ".git"
        || lib.hasPrefix "result" baseName
        || baseName == "naste-server"
        || baseName == "naste"
        || baseName == "tmp-data"
        || baseName == "test-data");
  };
in
{
  perSystem =
    { config, pkgs, ... }:
    {
      packages.default = pkgs.buildGoModule {
        pname = "naste-server";
        inherit src;
        version = "0.1.0";

        vendorHash = null;

        ldflags = [
          "-s"
          "-w"
        ];

        subPackages = [ "." ];
      };
    };
}
