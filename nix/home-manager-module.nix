{
  config,
  lib,
  pkgs,
  ...
}:
{
  options.services.naste-server = {
    enable = lib.mkEnableOption "naste paste service";

    package = lib.mkOption {
      type = lib.types.package;
      description = "naste-server package";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8080;
      description = "HTTP listen port";
    };

    dataDir = lib.mkOption {
      type = lib.types.path;
      default = "${config.home.homeDirectory}/.local/share/naste-server/data";
      description = "Storage directory for pastes";
    };

    maxPasteSize = lib.mkOption {
      type = lib.types.ints.positive;
      default = 10 * 1024 * 1024;
      description = "Maximum paste size in bytes (default: 10 MB)";
    };

    private = {
      user = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Username for private pastes (empty = public only)";
      };

      userFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = "File containing the username (overrides private.user)";
      };

      pass = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Password for private pastes";
      };

      passFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = "File containing the password (overrides private.pass)";
      };
    };
  };

  options.programs.naste-client = {
    enable = lib.mkEnableOption "naste CLI client";

    package = lib.mkOption {
      type = lib.types.package;
      description = "naste CLI client package";
    };

    endpoint = lib.mkOption {
      type = lib.types.str;
      description = "Paste server endpoint URL (e.g. http://localhost:8080 or https://paste.semi.sh)";
    };

    private = {
      user = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Username for private pastes (empty = public only)";
      };

      userFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = "File containing the username (overrides private.user)";
      };

      pass = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Password for private pastes";
      };

      passFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = "File containing the password (overrides private.pass)";
      };
    };
  };

  config =
    let
      cfg = config.services.naste-server;
      clientCfg = config.programs.naste-client;
    in
    lib.mkMerge [
      (lib.mkIf cfg.enable {
        systemd.user.services.naste-server = {
          Unit = {
            Description = "naste paste service";
            After = [ "network.target" ];
          };

          Install = {
            WantedBy = [ "default.target" ];
          };

          Service = {
            Type = "simple";
            ExecStart = lib.getExe' cfg.package "naste-server";
            Environment = [
              "PORT=${toString cfg.port}"
              "DATA_DIR=${cfg.dataDir}"
              "MAX_PASTE_SIZE=${toString cfg.maxPasteSize}"
            ]
            ++ lib.optional (
              cfg.private.user != "" && cfg.private.userFile == null
            ) "PRIVATE_USER=${cfg.private.user}"
            ++ lib.optional (
              cfg.private.pass != "" && cfg.private.passFile == null
            ) "PRIVATE_PASS=${cfg.private.pass}"
            ++ lib.optional (cfg.private.userFile != null) "PRIVATE_USER_FILE=${cfg.private.userFile}"
            ++ lib.optional (cfg.private.passFile != null) "PRIVATE_PASS_FILE=${cfg.private.passFile}";
            Restart = "on-failure";
            RestartSec = 5;
          };
        };
      })

      (lib.mkIf clientCfg.enable {
        home.packages = [ clientCfg.package ];
        home.sessionVariables.PASTE_ENDPOINT = clientCfg.endpoint;
      })

      (lib.mkIf (clientCfg.enable && clientCfg.private.user != "") {
        home.sessionVariables.PASTE_USER = clientCfg.private.user;
      })

      (lib.mkIf (clientCfg.enable && clientCfg.private.userFile != null) {
        home.sessionVariables.PASTE_USER_FILE = clientCfg.private.userFile;
      })

      (lib.mkIf (clientCfg.enable && clientCfg.private.pass != "") {
        home.sessionVariables.PASTE_PASS = clientCfg.private.pass;
      })

      (lib.mkIf (clientCfg.enable && clientCfg.private.passFile != null) {
        home.sessionVariables.PASTE_PASS_FILE = clientCfg.private.passFile;
      })

      {
        assertions = [
          {
            assertion = clientCfg.enable -> clientCfg.endpoint != "";
            message = "programs.naste-client.endpoint must be set when programs.naste-client.enable is true";
          }
        ];
      }
    ];
}
