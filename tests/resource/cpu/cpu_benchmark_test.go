package cpu_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/handler"
	"multi-tenant-domains/internal/middleware"
	"multi-tenant-domains/internal/model"
)

type benchStoreRepo struct{}

func (b *benchStoreRepo) CreateStore(ctx context.Context, store *model.Store) error { return nil }
func (b *benchStoreRepo) GetStoreByID(ctx context.Context, id uuid.UUID) (*model.Store, error) {
	cd := "bench.store.com"
	return &model.Store{
		ID:           id,
		Name:         "Bench Store",
		Subdomain:    "bench",
		CustomDomain: &cd,
		IsActive:     true,
	}, nil
}
func (b *benchStoreRepo) GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error) {
	return b.GetStoreByID(ctx, uuid.New())
}
func (b *benchStoreRepo) GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error) {
	return b.GetStoreByID(ctx, uuid.New())
}
func (b *benchStoreRepo) ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error) {
	return b.GetStoreByID(ctx, uuid.New())
}
func (b *benchStoreRepo) UpdateCustomDomainWithVerification(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID, customDomain string, token string, expiresAt time.Time) error {
	return nil
}
func (b *benchStoreRepo) UpdateDomainStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (b *benchStoreRepo) DeleteCustomDomain(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID) error {
	return nil
}
func (b *benchStoreRepo) ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error) {
	return nil, 0, nil
}
func (b *benchStoreRepo) ListStoresByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Store, error) {
	return nil, nil
}

// BenchmarkDomainValidation benchmarks Caddy on-demand ask endpoint CPU performance
func BenchmarkDomainValidation(b *testing.B) {
	log.SetOutput(io.Discard)
	repo := &benchStoreRepo{}
	cfg := &config.Config{
		BaseDomain:         "platform.local",
		AllowedSystemHosts: []string{"platform.local", "localhost"},
	}
	dh := handler.NewDomainHandler(repo, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/domains/validate?domain=bench.platform.local", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		dh.ValidateDomain(rr, req)
	}
}

// BenchmarkTenantMiddleware benchmarks hostname parsing and tenant resolution overhead
func BenchmarkTenantMiddleware(b *testing.B) {
	repo := &benchStoreRepo{}
	mw := middleware.TenantMiddleware(repo, "platform.local", []string{"localhost"})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := mw(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "bench.platform.local"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrapped.ServeHTTP(rr, req)
	}
}
