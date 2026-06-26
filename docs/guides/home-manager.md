# Home Manager

The Home Manager module runs `naste-server` as a **user** systemd service. No root required. Also installs the `naste` CLI client.

## Prerequisites

Add the flake input. The module includes its own packages, so no overlay is needed:

```nix
{
  inputs.naste.url = "github:semi710/naste";

  outputs = { self, home-manager, naste, ... }: {
    homeConfigurations.myuser = home-manager.lib.homeManagerConfiguration {
      modules = [
        naste.homeModules.default
        # ... service config here
      ];
    };
  };
}
```

## Quick Setup

```nix
{
  services.naste-server = {
    enable = true;
    port = 8080;
    private.user = "admin";
    private.pass = "secret";
  };

  programs.naste-client = {
    enable = true;
    endpoint = "https://paste.semi.sh";
  };
}
```

## With NixOS + Home Manager

If you use Home Manager as a NixOS module, add the HM module to `sharedModules`:

```nix
home-manager.sharedModules = [
  naste.homeModules.default
];
```

Then enable in your host config:

```nix
hm.services.naste-server = {
  enable = true;
  port = 8080;
  private.user = "admin";
  private.pass = "secret";
};
```

## With sops-nix

```nix
hm.services.naste-server = {
  enable = true;
  port = 8080;
  private.userFile = config.hm.sops.secrets."naste/user".path;
  private.passFile = config.hm.sops.secrets."naste/pass".path;
};
hm.sops.secrets."naste/user" = {
  sopsFile = ./secrets/server.yaml;
};
hm.sops.secrets."naste/pass" = {
  sopsFile = ./secrets/server.yaml;
};
```

!!! note "First deploy after adding secrets"
    After the first deploy with naste secrets, start a new SSH session. The `PASTE_USER_FILE` and `PASTE_PASS_FILE` session variables are set by home-manager at login. Existing sessions won't have them until you relogin.

## Options: services.naste-server

### services.naste-server.enable

Whether to enable the naste paste service as a user systemd service.

**Type:** boolean  
**Default:** `false`  
**Example:** `true`

### services.naste-server.port

HTTP listen port.

**Type:** port  
**Default:** `8080`  
**Example:** `9090`

### services.naste-server.dataDir

Storage directory for paste content.

**Type:** path  
**Default:** `~/.local/share/naste-server/data`  
**Example:** `/mnt/storage/naste`

!!! note "Data persistence"
    Pastes are stored as flat files and **never automatically purged**. No TTL or expiry.

### services.naste-server.maxPasteSize

Maximum paste size in bytes.

**Type:** positive integer  
**Default:** `10485760` (10 MB)  
**Example:** `52428800` (50 MB)

### services.naste-server.private.user

Username for private paste authentication.

**Type:** string  
**Default:** `""`  
**Example:** `"admin"`

### services.naste-server.private.pass

Password for private paste authentication.

**Type:** string  
**Default:** `""`  
**Example:** `"secret"`

### services.naste-server.private.userFile

Path to a file containing the username. Overrides `private.user`.

**Type:** null or path  
**Default:** `null`  
**Example:** `"/run/secrets/naste/user"`

### services.naste-server.private.passFile

Path to a file containing the password. Overrides `private.pass`.

**Type:** null or path  
**Default:** `null`  
**Example:** `"/run/secrets/naste/pass"`

## Options: programs.naste-client

### programs.naste-client.enable

Whether to install the naste CLI client to home packages and set session variables.

**Type:** boolean  
**Default:** `false`  
**Example:** `true`

### programs.naste-client.endpoint

Paste server endpoint URL. **Required** when enabled. The build fails with an assertion error if not set.

**Type:** string  
**Default:** none (assertion error if unset)  
**Example:** `"https://paste.semi.sh"`

### programs.naste-client.private.user

Username for private pastes. Sets the `PASTE_USER` session variable.

**Type:** string  
**Default:** `""`  
**Example:** `"admin"`

### programs.naste-client.private.userFile

Path to a file containing the username. Overrides `private.user`. Sets the `PASTE_USER_FILE` session variable.

**Type:** null or path  
**Default:** `null`  
**Example:** `"/run/secrets/naste/user"`

### programs.naste-client.private.pass

Password for private pastes. Sets the `PASTE_PASS` session variable.

**Type:** string  
**Default:** `""`  
**Example:** `"secret"`

### programs.naste-client.private.passFile

Path to a file containing the password. Overrides `private.pass`. Sets the `PASTE_PASS_FILE` session variable.

**Type:** null or path  
**Default:** `null`  
**Example:** `"/run/secrets/naste/pass"`

## Verifying

```bash
systemctl --user status naste-server
curl -s http://localhost:8080/health
journalctl --user -u naste-server -f
```

## Differences from NixOS Module

| Feature | NixOS Module | Home Manager Module |
|---------|-------------|---------------------|
| Service type | System service | User service |
| Requires root | Yes | No |
| Firewall | `openFirewall` option | Not available |
| User/group | Creates `naste` system user | Runs as your user |
| Data dir | `/var/lib/naste-server/data` | `~/.local/share/naste-server/data` |
| CLI client | `programs.naste-client` | `programs.naste-client` (same) |
| Env vars | `environment.variables` | `home.sessionVariables` |

## Real-world Example: ndots

[ndots](https://github.com/semi710/ndots) deploys the naste CLI client to all hosts via a shared home-manager module. Private credentials are added per-host via sops-nix.

### Shared client (all users)

`modules/home/naste.nix` - imported by the shared home module:

```nix
{ flake, ... }:
{
  imports = [ flake.inputs.naste.homeModules.default ];

  programs.naste-client = {
    enable = true;
    endpoint = "https://paste.semi.sh";
  };
}
```

No private credentials here. All users get the CLI with the endpoint pre-configured.

### Private credentials (hosts with sops)

```nix
# Standalone sops (mach, jp-mbp)
sops.secrets."naste/user" = { sopsFile = "${flake}/secrets/server.yaml"; };
sops.secrets."naste/pass" = { sopsFile = "${flake}/secrets/server.yaml"; };
programs.naste-client.private = {
  userFile = config.sops.secrets."naste/user".path;
  passFile = config.sops.secrets."naste/pass".path;
};
```

```nix
# NixOS + home-manager with hm shorthand (semi, dsd via workstation.nix)
hm.sops.secrets."naste/user".sopsFile = "${flake}/secrets/server.yaml";
hm.sops.secrets."naste/pass".sopsFile = "${flake}/secrets/server.yaml";
hm.programs.naste-client.private = {
  userFile = config.hm.sops.secrets."naste/user".path;
  passFile = config.hm.sops.secrets."naste/pass".path;
};
```

### Standalone home-manager (no sops)

Users in `hosts/home/` get public paste access only. No private credentials needed.
