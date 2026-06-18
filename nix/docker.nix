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
      packages.dockerImage =
        let
          binary = pkgs.buildGoModule {
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
        in
        pkgs.dockerTools.buildLayeredImage {
          name = "ghcr.io/semi710/naste-server";
          tag = "latest";
          config = {
            Cmd = [ "${binary}/bin/naste-server" ];
            ExposedPorts = {
              "8080/tcp" = { };
            };
            Env = [
              "PORT=8080"
              "DATA_DIR=/data/paste"
            ];
            User = "1000";
            WorkingDir = "/data/paste";
            Volumes = {
              "/data/paste" = { };
            };
          };
          extraCommands = ''
            mkdir -p data/paste/public data/paste/private data/paste/metadata
            chown -R 1000:1000 data/paste
          '';
        };
    };
}
