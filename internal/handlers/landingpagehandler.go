package handlers

import (
	"assignment-2/internal/utility"
	"net/http"
)

// The HTML/CSS in landingHTML was generated with help from Claude AI
// and reviewed by the group
const landingHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>EnvDash API</title>
<style>
  :root {
    --bg: #0f172a;
    --card: #1e293b;
    --accent: #38bdf8;
    --accent-soft: #0ea5e9;
    --text: #e2e8f0;
    --muted: #94a3b8;
    --border: #334155;
    --code-bg: #0b1220;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
    color: var(--text);
    min-height: 100vh;
    padding: 2rem 1rem;
    line-height: 1.6;
  }
  .container { max-width: 900px; margin: 0 auto; }
  header {
    text-align: center;
    padding: 3rem 1rem 2rem;
  }
  h1 {
    font-size: 2.75rem;
    font-weight: 700;
    background: linear-gradient(90deg, var(--accent), #a78bfa);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    margin-bottom: 0.5rem;
  }
  .tagline { color: var(--muted); font-size: 1.1rem; }
  .badge {
    display: inline-block;
    background: rgba(56, 189, 248, 0.15);
    color: var(--accent);
    padding: 0.25rem 0.75rem;
    border-radius: 999px;
    font-size: 0.85rem;
    margin-top: 1rem;
    border: 1px solid rgba(56, 189, 248, 0.3);
  }
  .endpoint-group {
    margin-bottom: 1.5rem;
    padding: 1rem;
    background: rgba(15, 23, 42, 0.35);
    border: 1px solid var(--border);
    border-radius: 10px;
  }
  .endpoint-group:last-child {
    margin-bottom: 0;
  }
  .endpoint-group h3 {
    color: #cbd5e1;
    font-size: 1rem;
    margin-bottom: 0.75rem;
    font-weight: 600;
  }
  section {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1.75rem;
    margin-bottom: 1.25rem;
  }
  h2 {
    font-size: 1.25rem;
    margin-bottom: 1rem;
    color: var(--accent);
  }
  .endpoint {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border);
    font-family: "SF Mono", Menlo, monospace;
    font-size: 0.95rem;
  }
  .endpoint:last-child { border-bottom: none; }
  .method {
    display: inline-block;
    padding: 0.2rem 0.55rem;
    border-radius: 4px;
    font-weight: 600;
    font-size: 0.75rem;
    min-width: 55px;
    text-align: center;
  }
  .method.get    { background: #065f46; color: #6ee7b7; }
  .method.post   { background: #1e3a8a; color: #93c5fd; }
  .method.put    { background: #78350f; color: #fcd34d; }
  .method.delete { background: #7f1d1d; color: #fca5a5; }
  .method.head   { background: #4c1d95; color: #c4b5fd; }
  .method.patch {  background: #0f766e;color: #99f6e4;}
  .path { color: var(--text); }
  .desc { color: var(--muted); margin-left: auto; font-family: sans-serif; font-size: 0.85rem; }
  code {
    background: var(--code-bg);
    padding: 0.15rem 0.4rem;
    border-radius: 4px;
    font-family: "SF Mono", Menlo, monospace;
    font-size: 0.9em;
    color: var(--accent);
  }
  footer {
    text-align: center;
    color: var(--muted);
    font-size: 0.85rem;
    margin-top: 2rem;
    padding: 1rem;
  }
  a { color: var(--accent); text-decoration: none; }
  a:hover { text-decoration: underline; }
  @media (max-width: 600px) {
    h1 { font-size: 2rem; }
    .endpoint { flex-wrap: wrap; }
    .desc { margin-left: 0; width: 100%; }
  }
</style>
</head>
<body>
  <div class="container">
    <header>
      <h1> EnvDash API</h1>
      <p class="tagline">Environment &amp; Air Quality Dashboard Service</p>
      <span class="badge">v1 · Running</span>
    </header>

    <section>
      <h2>About</h2>
      <p>A REST service aggregating live environmental data — temperature, precipitation, air quality, and country info — from multiple open APIs. Configurable dashboards with webhook-based notifications for lifecycle events and threshold alerts.</p>
    </section>

    <section>
  <h2>Endpoints</h2>

  <div class="endpoint-group">
    <h3>Registrations</h3>

    <div class="endpoint">
      <span class="method post">POST</span>
      <span class="path">/envdash/v1/registrations/</span>
      <span class="desc">Create configuration</span>
    </div>
    <div class="endpoint">
      <span class="method get">GET</span>
      <span class="path">/envdash/v1/registrations/{id}</span>
      <span class="desc">Get configuration</span>
    </div>
    <div class="endpoint">
      <span class="method head">HEAD</span>
      <span class="path">/envdash/v1/registrations/</span>
      <span class="desc">Headers only</span>
    </div>
    <div class="endpoint">
      <span class="method put">PUT</span>
      <span class="path">/envdash/v1/registrations/{id}</span>
      <span class="desc">Update configuration</span>
    </div>
    <div class="endpoint">
      <span class="method patch">PATCH</span>
      <span class="path">/envdash/v1/registrations/{id}</span>
      <span class="desc">Partial update</span>
    </div>
    <div class="endpoint">
      <span class="method delete">DELETE</span>
      <span class="path">/envdash/v1/registrations/{id}</span>
      <span class="desc">Delete configuration</span>
    </div>
  </div>

  <div class="endpoint-group">
    <h3>Dashboards</h3>

    <div class="endpoint">
      <span class="method get">GET</span>
      <span class="path">/envdash/v1/dashboards/{id}</span>
      <span class="desc">Get populated dashboard</span>
    </div>
  </div>

  <div class="endpoint-group">
    <h3>Notifications</h3>

    <div class="endpoint">
      <span class="method post">POST</span>
      <span class="path">/envdash/v1/notifications/</span>
      <span class="desc">Register a new webhook</span>
    </div>
    <div class="endpoint">
      <span class="method get">GET</span>
      <span class="path">/envdash/v1/notifications/{id}</span>
      <span class="desc">Retrieve a webhook registration</span>
    </div>
    <div class="endpoint">
      <span class="method get">GET</span>
      <span class="path">/envdash/v1/notifications/</span>
      <span class="desc">List all registered webhooks</span>
    </div>
    <div class="endpoint">
      <span class="method delete">DELETE</span>
      <span class="path">/envdash/v1/notifications/{id}</span>
      <span class="desc">Delete a webhook registration</span>
    </div>
  </div>

  <div class="endpoint-group">
    <h3>Authentication</h3>

    <div class="endpoint">
      <span class="method post">POST</span>
      <span class="path">/envdash/v1/auth/</span>
      <span class="desc">Register and get API key</span>
    </div>
    <div class="endpoint">
      <span class="method delete">DELETE</span>
      <span class="path">/envdash/v1/auth/{key}</span>
      <span class="desc">Revoke API key</span>
    </div>
  </div>

  <div class="endpoint-group">
    <h3>Status</h3>

    <div class="endpoint">
      <span class="method get">GET</span>
      <span class="path">/status/</span>
      <span class="desc">Health check</span>
    </div>
  </div>
</section>

    <section>
      <h2>Quick Start</h2>
      <p>Try it: <a href="/envdash/v1/status/">status</a></p>
    </section>

    <footer>
      Built for PROG2005 · NTNU Gjøvik · Spring 2026
    </footer>
  </div>
</body>
</html>`

// HandleLanding serves the root landing page.
func (h *Handler) HandleLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set(utility.ContentType, "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(landingHTML))
}
