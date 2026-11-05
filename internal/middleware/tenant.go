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

// TenantMiddleware resolves the tenant store from the incoming Host header and stores it in context.
func TenantMiddleware(repo repository.StoreRepository, baseDomain string, systemHosts []string) func(http.Handler) http.Handler {
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

			store, err := repo.ResolveStoreByHost(r.Context(), cleanedHost, baseDomain)
			if err != nil {
				if errors.Is(err, repository.ErrStoreNotFound) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":   "Tenant store not found",
						"host":    cleanedHost,
						"message": "No active store is configured for this domain or subdomain.",
					})
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "Failed to resolve tenant store",
					"details": err.Error(),
				})
				return
			}

			if !store.IsActive {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":    "Store inactive",
					"store_id": store.ID,
					"message":  "This store has been temporarily disabled.",
				})
				return
			}

			// Tenant successfully resolved; attach to context
			ctx := context.WithValue(r.Context(), tenantContextKey, store)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// StorefrontOnlyMiddleware enforces that a valid tenant MUST be present, returning 404 if not found.
func StorefrontOnlyMiddleware(repo repository.StoreRepository, baseDomain string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawHost := r.Host
			cleanedHost := model.CleanDomain(rawHost)

			store, err := repo.ResolveStoreByHost(r.Context(), cleanedHost, baseDomain)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "Store not found",
					"host":    cleanedHost,
					"message": "The requested domain or subdomain does not correspond to an active tenant store.",
				})
				return
			}

			if !store.IsActive {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":    "Store inactive",
					"store_id": store.ID,
					"message":  "This store has been temporarily disabled.",
				})
				return
			}

			ctx := context.WithValue(r.Context(), tenantContextKey, store)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WithTenant returns a new context with the tenant store attached.
func WithTenant(ctx context.Context, store *model.Store) context.Context {
	return context.WithValue(ctx, tenantContextKey, store)
}

// TenantFromContext extracts the tenant Store from request context.
func TenantFromContext(ctx context.Context) (*model.Store, bool) {
	val := ctx.Value(tenantContextKey)
	if val == nil {
		return nil, false
	}
	store, ok := val.(*model.Store)
	return store, ok
}

// MustGetTenant extracts the tenant Store from request context or panics if not present.
func MustGetTenant(ctx context.Context) *model.Store {
	store, ok := TenantFromContext(ctx)
	if !ok || store == nil {
		panic("middleware: tenant not found in request context")
	}
	return store
}
