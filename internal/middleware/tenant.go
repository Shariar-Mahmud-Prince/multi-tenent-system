package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
)

type contextKey string

const (
	tenantContextKey contextKey = "tenant_store"
)

// TenantMiddleware resolves the tenant application workspace from the incoming Host header and stores it in context.
func TenantMiddleware(repo repository.TenantRepository, baseDomain string, systemHosts []string) func(http.Handler) http.Handler {
	systemHostsMap := make(map[string]bool)
	for _, h := range systemHosts {
		systemHostsMap[strings.ToLower(strings.TrimSpace(h))] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawHost := r.Host
			cleanedHost := model.CleanDomain(rawHost)

			// If this is an allowed system/management host, allow request to pass through
			if systemHostsMap[cleanedHost] {
				next.ServeHTTP(w, r)
				return
			}

			tenant, err := repo.ResolveTenantByHost(r.Context(), cleanedHost, baseDomain)
			if err != nil {
				if errors.Is(err, repository.ErrTenantNotFound) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":   "Tenant workspace not found",
						"host":    cleanedHost,
						"message": "No active tenant workspace is configured for this domain or subdomain.",
					})
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "Failed to resolve tenant workspace",
					"details": err.Error(),
				})
				return
			}

			if !tenant.IsActive {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":     "Tenant workspace inactive",
					"tenant_id": tenant.ID,
					"message":   "This tenant workspace has been temporarily suspended or disabled.",
				})
				return
			}

			// Tenant successfully resolved; attach to context
			ctx := context.WithValue(r.Context(), tenantContextKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantOnlyMiddleware enforces that a valid tenant workspace MUST be present, returning 404 if not found.
func TenantOnlyMiddleware(repo repository.TenantRepository, baseDomain string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawHost := r.Host
			cleanedHost := model.CleanDomain(rawHost)

			tenant, err := repo.ResolveTenantByHost(r.Context(), cleanedHost, baseDomain)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "Tenant workspace not found",
					"host":    cleanedHost,
					"message": "The requested domain or subdomain does not correspond to an active tenant workspace.",
				})
				return
			}

			if !tenant.IsActive {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":     "Tenant workspace inactive",
					"tenant_id": tenant.ID,
					"message":   "This tenant workspace has been temporarily suspended or disabled.",
				})
				return
			}

			ctx := context.WithValue(r.Context(), tenantContextKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// StorefrontOnlyMiddleware is a backward-compatible alias.
var StorefrontOnlyMiddleware = TenantOnlyMiddleware

// WithTenant returns a new context with the tenant workspace attached.
func WithTenant(ctx context.Context, tenant *model.Tenant) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenant)
}

// TenantFromContext extracts the Tenant from request context.
func TenantFromContext(ctx context.Context) (*model.Tenant, bool) {
	val := ctx.Value(tenantContextKey)
	if val == nil {
		return nil, false
	}
	tenant, ok := val.(*model.Tenant)
	return tenant, ok
}

// MustGetTenant extracts the Tenant from request context or panics if not present.
func MustGetTenant(ctx context.Context) *model.Tenant {
	tenant, ok := TenantFromContext(ctx)
	if !ok || tenant == nil {
		panic("middleware: tenant not found in request context")
	}
	return tenant
}
