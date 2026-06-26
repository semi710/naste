# API Reference

All endpoints accept and return raw text or JSON. No authentication required for public pastes.

## Create Paste

```http
POST /api/paste
```

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `X-Slug` | No | Custom slug (random if not provided) |
| `X-Private` | No | Set to `true` for private paste |
| `X-Lang` | No | Language for syntax highlighting |

### Body

Raw text content of the paste.

### Response: 201 Created

```json
{
  "url": "https://paste.semi.sh/abc123"
}
```

For private pastes:

```json
{
  "url": "https://paste.semi.sh/private/abc123"
}
```

### Response: 409 Conflict

```json
{
  "error": "slug_exists"
}
```

### Response: 403 Forbidden

```json
{
  "error": "private_auth_not_configured"
}
```

Returned when `X-Private: true` is sent but the server has no `PRIVATE_USER`/`PRIVATE_PASS` configured.

### Examples

```bash
# Create paste
curl -X POST https://paste.semi.sh/api/paste -d "hello world"

# With custom slug
curl -X POST https://paste.semi.sh/api/paste \
  -H "X-Slug: mycode" \
  -d "content here"

# Private paste
curl -X POST https://paste.semi.sh/api/paste \
  -H "X-Private: true" \
  -H "X-Lang: go" \
  -d "package main"

# From file
curl -X POST https://paste.semi.sh/api/paste \
  -H "X-Slug: deploy" \
  -H "X-Lang: bash" \
  --data-binary @deploy.sh
```

## Overwrite Paste

```http
PUT /api/paste/{slug}
```

Replaces the content of an existing paste. For private pastes, requires Basic Auth.

### Response: 200 OK

```json
{
  "status": "updated"
}
```

### Response: 401 Unauthorized

```
WWW-Authenticate: Basic realm="Private Paste"
```

### Examples

```bash
# Overwrite public paste
curl -X PUT https://paste.semi.sh/api/paste/mycode -d "updated content"

# Overwrite private paste (requires auth)
curl -X PUT -u admin:secret https://paste.semi.sh/api/paste/secrets -d "new secrets"
```

## Get Public Paste

```http
GET /{slug}
```

### Browser (HTML)

Returns an HTML page with syntax highlighting (dark theme). The `User-Agent` header is checked to determine if the request is from a browser.

### curl / tools (raw text)

Returns raw text with `Content-Type: text/plain`.

### Force raw text

```http
GET /{slug}?raw=1
```

Always returns raw text regardless of User-Agent.

### Response: 200 OK

Browser: `Content-Type: text/html`
curl: `Content-Type: text/plain; charset=utf-8`

### Response: 404 Not Found

Paste does not exist or is private.

### Examples

```bash
# Browser: HTML with syntax highlighting
curl -s https://paste.semi.sh/abc123 | head

# Force raw text
curl -s https://paste.semi.sh/abc123?raw=1
```

## Get Private Paste

```http
GET /private/{slug}
```

Returns HTML with syntax highlighting for browsers, raw text for curl/tools. Requires HTTP Basic Auth.

### Browser (HTML)

Returns an HTML page with syntax highlighting (dark theme). The `User-Agent` header is checked to determine if the request is from a browser.

### curl / tools (raw text)

Returns raw text with `Content-Type: text/plain`.

### Force raw text

```http
GET /private/{slug}?raw=1
```

Always returns raw text regardless of User-Agent.

### Response: 200 OK

Browser: `Content-Type: text/html`
curl: `Content-Type: text/plain; charset=utf-8`

### Response: 401 Unauthorized

```http
WWW-Authenticate: Basic realm="Private Paste"
```

### Examples

```bash
# Browser: HTML with syntax highlighting (prompts for credentials)
curl -u admin:secret https://paste.semi.sh/private/secrets

# Force raw text
curl -u admin:secret https://paste.semi.sh/private/secrets?raw=1
```

## Health Check

```http
GET /health
```

### Response: 200 OK

```json
{
  "status": "ok"
}
```

### Examples

```bash
curl -s https://paste.semi.sh/health
```

## Landing Page

```http
GET /
```

Returns a minimal HTML page with usage instructions and a link to the source repository.

## Error Responses

All errors return JSON with an `error` field:

| Status | Error | Cause |
|--------|-------|-------|
| 400 | `invalid_slug` | Slug contains invalid characters or is reserved |
| 403 | `private_auth_not_configured` | Private paste requested but server has no auth |
| 404 | (empty) | Paste not found |
| 409 | `slug_exists` | Slug already in use in the same scope (public or private) |
| 413 | (empty) | Paste exceeds 10 MB limit |
| 500 | (empty) | Server error |

## Slug Validation

Slugs must:

- Be 1-64 characters long
- Contain only `[a-zA-Z0-9_-]`
- Not be a reserved word: `private`, `api`, `health`
- Not contain path traversal sequences (`..`, `/`, `\`)
