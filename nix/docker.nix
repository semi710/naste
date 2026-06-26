{ lib, ... }:
{
  perSystem =
    { pkgs, self', ... }:
    {
      packages.dockerImage =
        let
          binary = self'.packages.naste-server;
        in
        pkgs.dockerTools.buildLayeredImage {
          name = "ghcr.io/semi710/naste-server";
          tag = "latest";
          config = {
            Cmd = [ (lib.getExe' binary "naste-server") ];
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
          '';
        };
    };
}
