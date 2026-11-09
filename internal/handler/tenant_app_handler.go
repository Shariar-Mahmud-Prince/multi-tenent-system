package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"

	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/middleware"
)

// TenantAppHandler serves the active tenant web application instance.
type TenantAppHandler struct {
	cfg *config.Config
}

// Backward compatibility alias
type StorefrontHandler = TenantAppHandler

// NewTenantAppHandler creates a new TenantAppHandler.
func NewTenantAppHandler(cfg *config.Config) *TenantAppHandler {
	return &TenantAppHandler{
		cfg: cfg,
	}
}

// NewStorefrontHandler is a backward-compatible constructor.
func NewStorefrontHandler(cfg *config.Config) *StorefrontHandler {
	return NewTenantAppHandler(cfg)
}

// ServeTenantApp renders the application workspace for the resolved tenant.
func (h *TenantAppHandler) ServeTenantApp(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok || tenant == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Tenant context not available. Ensure request is addressed to a valid tenant domain.",
		})
		return
	}

	// If the client requested JSON, return structured tenant metadata
	if strings.Contains(r.Header.Get("Accept"), "application/json") || r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tenant": map[string]interface{}{
				"id":            tenant.ID,
				"name":          tenant.Name,
				"subdomain":     tenant.Subdomain,
				"subdomain_url": tenant.FullSubdomain(h.cfg.BaseDomain),
				"custom_domain": tenant.CustomDomain,
				"is_active":     tenant.IsActive,
			},
			"request": map[string]interface{}{
				"host":       r.Host,
				"path":       r.URL.Path,
				"user_agent": r.UserAgent(),
				"remote_ip":  r.RemoteAddr,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			},
			"saas_platform": map[string]interface{}{
				"version":    "1.0.0",
				"framework":  "Go + React Multi-Tenant SaaS Engine",
				"isolation":  "Logical Tenant Isolation with Server-Side Guard",
			},
		})
		return
	}

	// Otherwise, render a clean modern SaaS Tenant App Workspace view
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	tmpl := template.Must(template.New("tenant_app").Parse(tenantAppHTMLTemplate))
	customDomainDisplay := "None configured"
	if tenant.CustomDomain != nil && *tenant.CustomDomain != "" {
		customDomainDisplay = *tenant.CustomDomain
	}

	data := struct {
		TenantName   string
		TenantID     string
		Subdomain    string
		FullDomain   string
		CustomDomain string
		HostHeader   string
		CurrentTime  string
	}{
		TenantName:   tenant.Name,
		TenantID:     tenant.ID.String(),
		Subdomain:    tenant.Subdomain,
		FullDomain:   tenant.FullSubdomain(h.cfg.BaseDomain),
		CustomDomain: customDomainDisplay,
		HostHeader:   r.Host,
		CurrentTime:  time.Now().UTC().Format(time.RFC1123),
	}

	_ = tmpl.Execute(w, data)
}

// ServeStorefront is a backward-compatible alias.
func (h *TenantAppHandler) ServeStorefront(w http.ResponseWriter, r *http.Request) {
	h.ServeTenantApp(w, r)
}

const tenantAppHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.TenantName}} - Multi-Tenant SaaS Application</title>
    <style>
        :root {
            --primary: #3b82f6;
            --bg: #0f172a;
            --card-bg: #1e293b;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
            --accent: #10b981;
            --border: #334155;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background-color: var(--bg);
            color: var(--text-main);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
        }
        header {
            background-color: rgba(30, 41, 59, 0.8);
            backdrop-filter: blur(8px);
            border-bottom: 1px solid var(--border);
            padding: 1rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .brand { font-size: 1.25rem; font-weight: 700; color: #fff; }
        .tag {
            background-color: rgba(16, 185, 129, 0.1);
            color: var(--accent);
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 600;
        }
        main {
            flex: 1;
            max-width: 900px;
            margin: 2rem auto;
            padding: 0 1rem;
            width: 100%;
        }
        .card {
            background-color: var(--card-bg);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 2rem;
            margin-bottom: 1.5rem;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        }
        h1 { font-size: 1.75rem; margin-bottom: 0.5rem; }
        p { color: var(--text-muted); margin-bottom: 1.5rem; font-size: 0.95rem; }
        .meta-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-top: 1.5rem;
        }
        .meta-item {
            background-color: rgba(15, 23, 42, 0.5);
            padding: 1rem;
            border-radius: 8px;
            border: 1px solid var(--border);
        }
        .meta-label { font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase; margin-bottom: 0.25rem; }
        .meta-val { font-size: 0.9rem; font-weight: 500; word-break: break-all; }
        footer {
            border-top: 1px solid var(--border);
            padding: 1.5rem;
            text-align: center;
            color: var(--text-muted);
            font-size: 0.85rem;
        }
    </style>
</head>
<body>
    <header>
        <div class="brand">{{.TenantName}}</div>
        <div class="tag">ACTIVE TENANT WORKSPACE</div>
    </header>
    <main>
        <div class="card">
            <h1>{{.TenantName}}</h1>
            <p>Live SaaS application workspace resolved via custom domain or platform subdomain.</p>
            
            <div class="meta-grid">
                <div class="meta-item">
                    <div class="meta-label">Tenant UUID</div>
                    <div class="meta-val">{{.TenantID}}</div>
                </div>
                <div class="meta-item">
                    <div class="meta-label">Platform Subdomain</div>
                    <div class="meta-val">{{.FullDomain}}</div>
                </div>
                <div class="meta-item">
                    <div class="meta-label">Custom Domain</div>
                    <div class="meta-val">{{.CustomDomain}}</div>
                </div>
                <div class="meta-item">
                    <div class="meta-label">Request Host</div>
                    <div class="meta-val">{{.HostHeader}}</div>
                </div>
            </div>
        </div>
    </main>
    <footer>
        Multi-Tenant SaaS Foundation Architecture &bull; Generated: {{.CurrentTime}}
    </footer>
</body>
</html>
`
