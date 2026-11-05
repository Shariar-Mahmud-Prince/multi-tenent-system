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

// StorefrontHandler serves the public tenant storefront.
type StorefrontHandler struct {
	cfg *config.Config
}

// NewStorefrontHandler creates a new StorefrontHandler.
func NewStorefrontHandler(cfg *config.Config) *StorefrontHandler {
	return &StorefrontHandler{
		cfg: cfg,
	}
}

// ServeStorefront renders the storefront representation for the resolved tenant.
func (h *StorefrontHandler) ServeStorefront(w http.ResponseWriter, r *http.Request) {
	store, ok := middleware.TenantFromContext(r.Context())
	if !ok || store == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Store context not available. Ensure request is addressed to a valid tenant domain.",
		})
		return
	}

	// If the client requested JSON (or path ends in .json), return structured JSON
	if strings.Contains(r.Header.Get("Accept"), "application/json") || r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"store": map[string]interface{}{
				"id":            store.ID,
				"name":          store.Name,
				"subdomain":     store.Subdomain,
				"subdomain_url": store.FullSubdomain(h.cfg.BaseDomain),
				"custom_domain": store.CustomDomain,
				"is_active":     store.IsActive,
			},
			"request": map[string]interface{}{
				"host":       r.Host,
				"path":       r.URL.Path,
				"user_agent": r.UserAgent(),
				"remote_ip":  r.RemoteAddr,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			},
			"catalog": []map[string]interface{}{
				{"id": 1, "title": "Standard Subscription", "price": "$29.00"},
				{"id": 2, "title": "Enterprise Edition", "price": "$199.00"},
			},
		})
		return
	}

	// Otherwise, render a clean modern HTML storefront
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	tmpl := template.Must(template.New("storefront").Parse(storefrontHTMLTemplate))
	customDomainDisplay := "None configured"
	if store.CustomDomain != nil && *store.CustomDomain != "" {
		customDomainDisplay = *store.CustomDomain
	}

	data := struct {
		StoreName    string
		StoreID      string
		Subdomain    string
		FullDomain   string
		CustomDomain string
		HostHeader   string
		CurrentTime  string
	}{
		StoreName:    store.Name,
		StoreID:      store.ID.String(),
		Subdomain:    store.Subdomain,
		FullDomain:   store.FullSubdomain(h.cfg.BaseDomain),
		CustomDomain: customDomainDisplay,
		HostHeader:   r.Host,
		CurrentTime:  time.Now().UTC().Format(time.RFC1123),
	}

	_ = tmpl.Execute(w, data)
}

const storefrontHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.StoreName}} - Multi-Tenant Storefront</title>
    <style>
        :root {
            --primary: #2563eb;
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
            justify-content: center;
            align-items: center;
            padding: 2rem;
        }
        .container {
            width: 100%;
            max-width: 640px;
            background: var(--card-bg);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 2.5rem;
            box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.4), 0 8px 10px -6px rgba(0, 0, 0, 0.3);
        }
        .badge {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            background-color: rgba(16, 185, 129, 0.15);
            color: var(--accent);
            padding: 4px 10px;
            border-radius: 9999px;
            margin-bottom: 1.5rem;
        }
        .badge::before {
            content: "";
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background-color: var(--accent);
        }
        h1 {
            font-size: 2rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
            color: #ffffff;
        }
        p.subtitle {
            color: var(--text-muted);
            font-size: 1rem;
            margin-bottom: 2rem;
        }
        .details-grid {
            display: grid;
            grid-template-columns: 1fr;
            gap: 1rem;
            background: #0f172a;
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 1.25rem;
            margin-bottom: 2rem;
        }
        .detail-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0.25rem 0;
            font-size: 0.9rem;
            border-bottom: 1px solid #1e293b;
        }
        .detail-row:last-child { border-bottom: none; }
        .label { color: var(--text-muted); }
        .value { font-family: monospace; font-weight: 600; color: #38bdf8; }
        .footer {
            text-align: center;
            font-size: 0.8rem;
            color: var(--text-muted);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="badge">Multi-Tenant Routing Active</div>
        <h1>{{.StoreName}}</h1>
        <p class="subtitle">Live storefront successfully resolved via dynamic multi-tenant domain routing.</p>
        
        <div class="details-grid">
            <div class="detail-row">
                <span class="label">Store UUID</span>
                <span class="value">{{.StoreID}}</span>
            </div>
            <div class="detail-row">
                <span class="label">Platform Subdomain</span>
                <span class="value">{{.FullDomain}}</span>
            </div>
            <div class="detail-row">
                <span class="label">Custom Domain</span>
                <span class="value">{{.CustomDomain}}</span>
            </div>
            <div class="detail-row">
                <span class="label">Matched Host Header</span>
                <span class="value">{{.HostHeader}}</span>
            </div>
            <div class="detail-row">
                <span class="label">Server UTC Time</span>
                <span class="value">{{.CurrentTime}}</span>
            </div>
        </div>

        <div class="footer">
            Powered by Go &bull; Caddy On-Demand TLS &bull; PostgreSQL
        </div>
    </div>
</body>
</html>`
