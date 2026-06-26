{ lib, ... }:
let
  src = lib.cleanSourceWith {
    src = ../.;
    filter =
      path: _type:
      let
        baseName = baseNameOf path;
      in
      !(
        baseName == ".direnv"
        || baseName == ".git"
        || lib.hasPrefix "result" baseName
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
  perSystem =
    { pkgs, self', ... }:
    {
      packages = {
        default = self'.packages.naste;
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

      apps.naste.program = lib.getExe self'.packages.naste;

      apps.docs =
        let
          env = pkgs.python3.withPackages (
            ps: with ps; [
              mkdocs
              mkdocs-material
            ]
          );
        in
        {
          type = "app";
          program = "${env}/bin/mkdocs";
        };
    };
}
