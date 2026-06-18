package handlers

import "net/http"

// isBrowser returns true if the request appears to come from a web browser.
func isBrowser(r *http.Request) bool {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return false
	}
	// Simple heuristic: browsers include Mozilla or common engine strings
	return containsAny(ua, []string{"Mozilla", "Chrome", "Safari", "Firefox", "Edge", "Opera"})
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
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
<title>paste.semi.sh</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:system-ui,-apple-system,sans-serif;line-height:1.6;color:#333;max-width:800px;margin:0 auto;padding:2rem;background:#fafafa}
h1{font-size:2.5rem;margin-bottom:.5rem;color:#222}
p.lead{color:#666;margin-bottom:2rem;font-size:1.1rem}
section{margin-bottom:2.5rem}
h2{font-size:1.3rem;margin-bottom:1rem;color:#444;border-bottom:2px solid #ddd;padding-bottom:.3rem}
pre{background:#1e1e1e;color:#d4d4d4;padding:1rem;border-radius:6px;overflow-x:auto;font-size:.9rem;margin:.5rem 0}
pre .comment{color:#6a9955}
pre .kw{color:#569cd6}
pre .str{color:#ce9178}
code{background:#eee;padding:.15rem .4rem;border-radius:3px;font-size:.9rem}
.install{background:#2ea44f;color:#fff;padding:1rem;border-radius:6px;margin:1rem 0}
.install code{background:rgba(255,255,255,.15);color:#fff}
ul{padding-left:1.5rem;margin:.5rem 0}
li{margin:.3rem 0}
a{color:#0366d6;text-decoration:none}
a:hover{text-decoration:underline}
.footer{margin-top:3rem;padding-top:1rem;border-top:1px solid #ddd;color:#666;font-size:.9rem}
</style>
</head>
<body>
<h1>paste.semi.sh</h1>
<p class="lead">A minimal, self-hosted paste service for the command line.</p>

<section>
<h2>Quick Install</h2>
<div class="install">
<code>curl -fsSL https://paste.semi.sh/install | sh</code>
</div>
<p>This installs the <code>naste</code> CLI to <code>/usr/local/bin/</code>.</p>
</section>

<section>
<h2>Usage</h2>
<pre><span class="comment"># Pipe any text to get a public URL</span>
<span class="kw">cat</span> file.txt | naste
<span class="comment"># → https://paste.semi.sh/abc123</span>

<span class="comment"># Use a custom slug</span>
<span class="kw">cat</span> file.txt | naste --slug logs
<span class="comment"># → https://paste.semi.sh/logs</span>

<span class="comment"># Create a private paste</span>
<span class="kw">cat</span> file.txt | naste --private
<span class="comment"># → https://paste.semi.sh/private/xyz789</span>

<span class="comment"># Private paste with custom slug</span>
<span class="kw">cat</span> file.txt | naste --private --slug serverlog
</pre>
</section>

<section>
<h2>API</h2>
<p>All endpoints accept and return raw text or JSON.</p>
<pre>POST /api/paste         <span class="comment"># Create a new paste</span>
PUT  /api/paste/{slug}  <span class="comment"># Overwrite an existing paste</span>
GET  /{slug}            <span class="comment"># Retrieve a public paste</span>
GET  /private/{slug}    <span class="comment"># Retrieve a private paste (requires auth)</span>
GET  /health            <span class="comment"># Health check</span>
GET  /install           <span class="comment"># CLI install script</span>
</pre>
</section>

<section>
<h2>Features</h2>
<ul>
<li>No database — files only</li>
<li>Public and private pastes</li>
<li>Custom slugs</li>
<li>HTTP Basic Auth for private content</li>
<li>Single binary deployment</li>
<li>10 MB size limit</li>
</ul>
</section>

<section>
<h2>CLI Config</h2>
<p>Create <code>~/.config/naste/config.toml</code>:</p>
<pre>[paste]
endpoint = "https://paste.semi.sh"
user = "your_username"
password = "your_password"
</pre>
</section>

<div class="footer">
<a href="https://github.com/semi710/nastebin">source</a>
</div>
</body>
</html>
`
