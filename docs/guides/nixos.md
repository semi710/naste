# NixOS

The NixOS module runs `naste-server` as a hardened systemd service and optionally installs the `naste` CLI client.

## Prerequisites

Add the flake input and overlay to your system flake:

```nix
{
  inputs.naste.url = "github:semi710/naste";

  outputs = { self, nixpkgs, naste, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        naste.nixosModules.default
        {
          nixpkgs.overlays = [ naste.overlays.default ];
          # ... service config here
        }
      ];
    };
  };
}
```

The overlay provides `pkgs.naste-server` and `pkgs.naste`.

## Quick Setup

```nix
{
  services.naste-server = {
    enable = true;
    port = 8080;
    openFirewall = true;
    private.userFile = config.sops.secrets."naste/user".path;
    private.passFile = config.sops.secrets."naste/pass".path;
  };

  programs.naste-client = {
    enable = true;
    endpoint = "https://paste.semi.sh";
  };

  sops.secrets."naste/user" = {
    sopsFile = ./secrets/server.yaml;
    group = "naste";
    mode = "0440";
  };
  sops.secrets."naste/pass" = {
    sopsFile = ./secrets/server.yaml;
    group = "naste";
    mode = "0440";
  };
}
```

## Options: services.naste-server

### services.naste-server.enable

Whether to enable the naste paste service. Creates a `naste` system user/group and a hardened systemd service.

**Type:** boolean  
**Default:** `false`  
**Example:** `true`

### services.naste-server.port

HTTP listen port.

**Type:** port  
**Default:** `8080`  
**Example:** `9090`

### services.naste-server.dataDir

Storage directory for paste content. Created automatically, owned by the `naste` user.

**Type:** path  
**Default:** `/var/lib/naste-server/data`  
**Example:** `/mnt/storage/naste`

!!! note "Data persistence"
    Pastes are stored as flat files and **never automatically purged**. No TTL or expiry. Delete files manually or remove the `dataDir`.

### services.naste-server.maxPasteSize

Maximum paste size in bytes. Pastes exceeding this are rejected with HTTP 413.

**Type:** positive integer  
**Default:** `10485760` (10 MB)  
**Example:** `52428800` (50 MB)

### services.naste-server.private.user

Username for private paste authentication. Empty means public pastes only.

**Type:** string  
**Default:** `""`  
**Example:** `"admin"`

### services.naste-server.private.pass

Password for private paste authentication.

**Type:** string  
**Default:** `""`  
**Example:** `"secret"`

### services.naste-server.private.userFile

Path to a file containing the username. Overrides `private.user`. Use with sops-nix.

**Type:** null or path  
**Default:** `null`  
**Example:** `"/run/secrets/naste/user"`

### services.naste-server.private.passFile

Path to a file containing the password. Overrides `private.pass`. Use with sops-nix.

**Type:** null or path  
**Default:** `null`  
**Example:** `"/run/secrets/naste/pass"`

### services.naste-server.openFirewall

Whether to open the firewall for the configured port.

**Type:** boolean  
**Default:** `false`  
**Example:** `true`

## Options: programs.naste-client

### programs.naste-client.enable

Whether to install the naste CLI client and set the `PASTE_ENDPOINT` environment variable.

**Type:** boolean  
**Default:** `false`  
**Example:** `true`

### programs.naste-client.endpoint

Paste server endpoint URL. **Required** when enabled. The build fails with an assertion error if not set.

**Type:** string  
**Default:** none (assertion error if unset)  
**Example:** `"https://paste.semi.sh"`

### programs.naste-client.private.user

Username for private pastes. Sets the `PASTE_USER` environment variable.

**Type:** string  
**Default:** `""`  
**Example:** `"admin"`

### programs.naste-client.private.userFile

Path to a file containing the username. Overrides `private.user`. Sets the `PASTE_USER_FILE` environment variable.

**Type:** null or path  
**Default:** `null`  
**Example:** `"/run/secrets/naste/user"`

### programs.naste-client.private.pass

Password for private pastes. Sets the `PASTE_PASS` environment variable.

**Type:** string  
**Default:** `""`  
**Example:** `"secret"`

### programs.naste-client.private.passFile

Path to a file containing the password. Overrides `private.pass`. Sets the `PASTE_PASS_FILE` environment variable.

**Type:** null or path  
**Default:** `null`  
**Example:** `"/run/secrets/naste/pass"`

## sops-nix Integration

```nix
# secrets/server.yaml (encrypted with sops)
# naste:
#     user: admin
#     pass: your-secret-password

sops.secrets."naste/user" = {
  sopsFile = ./secrets/server.yaml;
  group = "naste";
  mode = "0440";
};
sops.secrets."naste/pass" = {
  sopsFile = ./secrets/server.yaml;
  group = "naste";
  mode = "0440";
};
```

Secrets are deployed to `/run/secrets/naste/user` and `/run/secrets/naste/pass`. The `naste` group has read access (mode 0440).

!!! note "First deploy after adding secrets"
    After the first deploy with naste secrets, start a new SSH session. The `PASTE_USER_FILE` and `PASTE_PASS_FILE` session variables are set by home-manager at login. Existing sessions won't have them until you relogin.

## Service Hardening

| Setting | Value |
|---------|-------|
| `NoNewPrivileges` | `true` |
| `ProtectHome` | `true` |
| `PrivateTmp` | `true` |
| `StateDirectory` | `naste-server` |
| `ReadWritePaths` | `dataDir` |
| `Restart` | `on-failure` |
| `RestartSec` | `5` |

!!! note "ProtectSystem removed"
    `ProtectSystem = "strict"` was removed because it created a mount namespace that blocked sops-nix secret reads from `/run/secrets/`. The service already runs as a dedicated user with `PrivateTmp` and `ProtectHome`.

## Verifying

```bash
systemctl status naste-server.service
curl -s http://localhost:8080/health
journalctl -u naste-server.service -f
```
