package handlers

import (
	"net/http"
	"strings"
)

// isBrowser returns true if the request appears to come from a web browser.
func isBrowser(r *http.Request) bool {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return false
	}
	for _, b := range []string{"Mozilla", "Chrome", "Safari", "Firefox", "Edge", "Opera"} {
		if strings.Contains(ua, b) {
			return true
		}
	}
	return false
}

const pasteViewHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<link rel="icon" href="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxMDAgMTAwIj48cmVjdCB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgcng9IjIwIiBmaWxsPSIjMTYxYjIyIi8+PHRleHQgeD0iNTAiIHk9IjY4IiBmb250LXNpemU9IjUwIiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBmaWxsPSIjMjJkM2VlIiBmb250LWZhbWlseT0ibW9ub3NwYWNlIiBmb250LXdlaWdodD0iYm9sZCI+TjwvdGV4dD48L3N2Zz4=">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'SF Mono',Monaco,'Courier New',monospace;font-size:15px;line-height:1.7;color:#e2e4e8;background:#0d1117;padding:0;margin:0}
.header{background:#161b22;padding:1rem 2rem;border-bottom:1px solid #30363d;display:flex;justify-content:space-between;align-items:center}
.header h1{font-size:1.1rem;color:#f0883e;font-weight:700;margin:0}
.header .meta{color:#8b949e;font-size:.85rem}
.header a{color:#58a6ff;text-decoration:none;font-weight:600}
.header a:hover{text-decoration:underline}
.raw-link{color:#58a6ff}
.content{padding:2rem;overflow-x:auto}
pre{margin:0;white-space:pre-wrap;word-wrap:break-word;font-size:15px}
.keyword{color:#ff0000 !important;font-weight:bold !important}
.string{color:#00ff00 !important;font-weight:bold !important}
.comment{color:#ffff00 !important;font-weight:bold !important;font-style:italic}
.number{color:#00ffff !important;font-weight:bold !important}
.function{color:#ff00ff !important;font-weight:bold !important}
.operator{color:#ff8800 !important;font-weight:bold !important}
</style>
</head>
<body>
<div class="header">
<h1>%s</h1>
<div class="meta">
<a href="?raw=1" class="raw-link">raw</a> &middot; %s
</div>
</div>
<div class="content">
<pre><code>%s</code></pre>
</div>
</body>
</html>`

const landingHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>naste | paste service</title>
<link rel="icon" href="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxMDAgMTAwIj48cmVjdCB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgcng9IjIwIiBmaWxsPSIjMTYxYjIyIi8+PHRleHQgeD0iNTAiIHk9IjY4IiBmb250LXNpemU9IjUwIiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBmaWxsPSIjMjJkM2VlIiBmb250LWZhbWlseT0ibW9ub3NwYWNlIiBmb250LXdlaWdodD0iYm9sZCI+TjwvdGV4dD48L3N2Zz4=">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'SF Mono',Monaco,'Courier New',monospace;font-size:15px;line-height:1.7;color:#e2e4e8;background:#0d1117;padding:2rem;max-width:800px;margin:0 auto;min-height:100vh}
h1{font-size:1.5rem;color:#22d3ee;margin-bottom:1.5rem;font-weight:700}
h2{font-size:1.1rem;color:#22d3ee;margin:2rem 0 1rem;font-weight:700}
p{color:#8b949e;margin-bottom:1rem}
a{color:#58a6ff;text-decoration:none}
a:hover{text-decoration:underline}
pre{background:#161b22;border:1px solid #30363d;border-radius:6px;padding:1rem;overflow-x:auto;margin:1rem 0;color:#d4d4d4}
.comment{color:#6a9955}
.kw{color:#569cd6}
.str{color:#ce9178}
.footer{margin-top:3rem;padding-top:1rem;border-top:1px solid #30363d;color:#8b949e;font-size:.85rem}
</style>
</head>
<body>
<h1>naste — minimal paste service</h1>

<p>A self-hosted paste service for the command line. No database. No frameworks. Just files. Host your own for privacy or provide it as a free service to others.</p>

<h2>Quick Start</h2>
<pre><span class="comment"># Pipe any text</span>
<span class="kw">echo</span> <span class="str">"hello world"</span> | naste
<span class="comment"># → https://your-domain.com/abc123</span>

<span class="comment"># With a custom slug</span>
<span class="kw">cat</span> deploy.sh | naste --slug deploy

<span class="comment"># Private paste</span>
<span class="kw">cat</span> secrets.env | naste --private

<span class="comment"># Install via go</span>
go install github.com/semi710/naste/cmd/naste@latest

<span class="comment"># Or run with nix (no install needed)</span>
echo "hello world" | nix run github:semi710/naste#naste --
</pre>

<h2>API Endpoints</h2>
<pre>POST /api/paste         <span class="comment"># Create paste</span>
GET  /{slug}            <span class="comment"># View paste (HTML in browser, raw with curl)</span>
GET  /{slug}?raw=1      <span class="comment"># Force raw text</span>
GET  /private/{slug}    <span class="comment"># Private paste (Basic Auth)</span>
GET  /health            <span class="comment"># Health check</span>
</pre>

<div class="footer">
<a href="https://naste.semi.sh">docs</a> &middot; <a href="https://github.com/semi710/naste">source on github</a>
</div>
</body>
</html>
`
