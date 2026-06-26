# Reverse Proxy

Put naste-server behind a reverse proxy for TLS, custom domains, and rate limiting.

## Caddy (Recommended)

Caddy provides automatic HTTPS via Let's Encrypt.

```caddyfile
paste.example.com {
    reverse_proxy localhost:8080

    # Rate limiting (requires caddy-ratelimit plugin)
    # rate_limit {
    #     zone naste {
    #         events 10
    #         window 1m
    #     }
    # }
}
```

Reload Caddy:

```bash
systemctl reload caddy
```

## Nginx

```nginx
server {
    listen 80;
    server_name paste.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name paste.example.com;

    ssl_certificate /etc/ssl/certs/paste.example.com.crt;
    ssl_certificate_key /etc/ssl/private/paste.example.com.key;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Max upload size (must match server's 10 MB limit)
        client_max_body_size 10m;
    }
}
```

## Tailscale Funnel

Expose naste-server without a domain or TLS certificate using Tailscale Funnel:

```bash
# Enable Tailscale funnel on port 8080
sudo tailscale funnel 8080

# Your server is now accessible at:
# https://your-machine.tailnet-name.ts.net
```

No reverse proxy config needed. Tailscale handles TLS automatically.

## Tailscale Serve (internal only)

If you only want Tailscale-network access (no public internet):

```bash
# Serve on Tailscale network only
tailscale serve 8080

# Accessible at:
# http://your-machine.tailnet-name.ts.net
```

## NixOS Caddy Module

If using NixOS with Caddy:

```nix
services.caddy = {
  enable = true;
  virtualHosts."paste.example.com".extraConfig = ''
    reverse_proxy localhost:8080
  '';
};
```

Caddy in NixOS runs as root by default and manages its own TLS certificates.

## Important Notes

- The server sets `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and CSP headers on all responses
- For paste views with syntax highlighting, the server overrides CSP to `style-src 'unsafe-inline'` (required for inline CSS colors)
- The server reads `r.Host` directly, not `X-Forwarded-Host`, to prevent header injection
- Set `client_max_body_size` to at least `10m` in Nginx to match the server's 10 MB limit
