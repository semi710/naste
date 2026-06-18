# Paste.semi.sh - Technical Implementation Plan

## Goal

Build a self-hosted paste service optimized for CLI usage.

Primary workflow:

```bash
cat file.txt | paste
```

Output:

```text
https://paste.semi.sh/abc123
```

The service supports:

* Public pastes
* Private pastes
* Custom slugs
* Basic authentication for private content
* Raw text serving
* Zero database
* Single binary deployment
* File-backed storage

This is not intended to be a full Pastebin clone.

No syntax highlighting.
No accounts.
No comments.
No sharing UI.
No markdown rendering.

The service should feel similar to paste.rs while adding private/public separation and custom slugs. Public/private separation via HTTP Basic Auth is a common recommendation for self-hosted paste services.

---

# Stack

Backend:

* Go 1.25+
* Standard library only where possible

Frontend:

* Minimal HTML templates
* No JS framework

Storage:

* Filesystem

Reverse Proxy:

* Caddy

Deployment:

* Systemd

---

# Domain Layout

```text
paste.semi.sh
```

Routes:

```text
GET  /
GET  /{slug}

GET  /private/{slug}

POST /api/paste

GET  /install
```

---

# Storage Layout

```text
/data/paste/

├── public/
│   ├── abc123
│   ├── logs
│   └── test

├── private/
│   ├── serverlog
│   └── dbdump

└── metadata/
```

No database.

Each file contains raw content.

Metadata stored separately.

Example:

```json
{
  "slug": "serverlog",
  "private": true,
  "created_at": "2026-06-12T12:00:00Z",
  "size": 14593,
  "ttl": "never"
}
```

---

# Public Paste Flow

Create:

```bash
cat logs.txt | paste
```

Request:

```http
POST /api/paste
```

Response:

```json
{
  "url": "https://paste.semi.sh/abc123"
}
```

Read:

```text
GET /abc123
```

Returns:

```text
raw content only
```

Content-Type:

```text
text/plain
```

---

# Custom Slugs

Create:

```bash
cat logs.txt | paste --slug serverlog
```

Request:

```json
{
  "slug": "serverlog"
}
```

If slug unused:

```http
201 Created
```

If slug exists:

```http
409 Conflict
```

Response:

```json
{
  "error": "slug_exists"
}
```

CLI then prompts:

```text
Slug exists.
Override? [y/N]
```

If user confirms:

```bash
paste --slug serverlog --force
```

Request:

```http
PUT /api/paste/serverlog
```

---

# Private Pastes

Create:

```bash
PASTE_USER=semi \
PASTE_PASS=supersecret \
cat logs.txt | paste --private --slug serverlog
```

Request:

```http
POST /api/paste
Authorization: Basic ...
```

Stored under:

```text
/private/serverlog
```

Generated URL:

```text
https://paste.semi.sh/private/serverlog
```

---

# Private Paste Viewing

Visiting:

```text
https://paste.semi.sh/private/serverlog
```

Without auth:

```http
401 Unauthorized
WWW-Authenticate: Basic realm="Private Paste"
```

Browser shows login prompt.

After successful login:

```text
raw file contents
```

Authentication should be global.

Environment:

```env
PRIVATE_USER=semi
PRIVATE_PASS=supersecret
```

No per-paste passwords.

---

# Authentication Rules

Public:

```text
GET /slug
```

No auth.

Private:

```text
GET /private/*
POST private paste
PUT private paste
DELETE private paste
```

Require auth.

---

# Slug Rules

Allowed:

```text
a-z
A-Z
0-9
-
_
```

Examples:

```text
logs
server-log
build_2026
```

Reject:

```text
../../etc/passwd
foo/bar
space here
```

Max length:

```text
64
```

Min length:

```text
1
```

Reserved:

```text
private
api
install
health
```

---

# Auto Generated Slugs

Default length:

```text
6
```

Alphabet:

```text
abcdefghijklmnopqrstuvwxyz
ABCDEFGHIJKLMNOPQRSTUVWXYZ
0123456789
```

Example:

```text
A7sdP2
```

Retry until unique.

---

# Install Script Endpoint

Landing page should advertise:

```bash
curl -fsSL https://paste.semi.sh/install | sh
```

Install script:

1. Detect OS
2. Download correct binary
3. Install to:

```text
/usr/local/bin/paste
```

4. Make executable

---

# CLI Tool

Command:

```bash
paste
```

Examples:

```bash
cat file.txt | paste

cat file.txt | paste --slug logs

cat file.txt | paste --private

cat file.txt | paste --private --slug serverlog

cat file.txt | paste --slug logs --force
```

Config file:

```text
~/.config/paste/config.toml
```

Example:

```toml
endpoint = "https://paste.semi.sh"

user = "semi"
password = "supersecret"
```

Environment variables override config.

---

# Landing Page

GET /

Displays:

* Service description
* Installation command
* Usage examples
* API examples
* Public/private explanation

No login page required.

---

# API Specification

Create:

```http
POST /api/paste
```

Headers:

```http
X-Slug: logs
X-Private: true
```

Body:

```text
raw text
```

Response:

```json
{
  "url": "https://paste.semi.sh/logs"
}
```

---

# Limits

Paste size:

```text
10 MB
```

Max request body:

```text
10 MB
```

Reject larger uploads.

Response:

```http
413 Payload Too Large
```

---

# Security

Implement:

* Request size limit
* Path traversal protection
* Slug validation
* Basic auth
* TLS via Caddy
* Atomic file writes
* Temporary file + rename pattern

Do not execute uploaded content.

Always serve:

```http
Content-Type: text/plain
```

This prevents accidental XSS.

---

# Observability

Endpoints:

```text
GET /health
```

Response:

```json
{
  "status": "ok"
}
```

Logs:

```text
upload
read
overwrite
delete
auth failure
```

---

# Future Features

Not required for v1.

Potential v2:

* TTL expiration
* Burn-after-read
* QR code generation
* Syntax highlighting
* S3 backend
* Multi-user support
* Search
* Paste deletion API

---

# Success Criteria

The following workflow must work:

```bash
cat server.log | paste
```

Returns:

```text
https://paste.semi.sh/A7sdP2
```

Then:

```bash
curl https://paste.semi.sh/A7sdP2
```

Returns original content.

And:

```bash
cat prod.log | paste --private --slug serverlog
```

Creates:

```text
https://paste.semi.sh/private/serverlog
```

Accessible only with configured Basic Auth credentials.
