package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"multi-tenant-domains/internal/middleware"
	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
)

type mockStoreRepo struct {
	store *model.Store
}

func (m *mockStoreRepo) CreateStore(ctx context.Context, store *model.Store) error { return nil }
func (m *mockStoreRepo) GetStoreByID(ctx context.Context, id uuid.UUID) (*model.Store, error) {
	return m.store, nil
}
func (m *mockStoreRepo) GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error) {
	if m.store != nil && m.store.Subdomain == subdomain {
		return m.store, nil
	}
	return nil, repository.ErrStoreNotFound
}
func (m *mockStoreRepo) GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error) {
	if m.store != nil && m.store.CustomDomain != nil && *m.store.CustomDomain == customDomain {
		return m.store, nil
	}
	return nil, repository.ErrStoreNotFound
}
func (m *mockStoreRepo) ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error) {
	cleaned := model.CleanDomain(host)
	if cleaned == "shop.acme.test" || cleaned == "acme.yourplatform.test" {
		if m.store != nil {
			return m.store, nil
		}
	}
	return nil, repository.ErrStoreNotFound
}
func (m *mockStoreRepo) UpdateCustomDomain(ctx context.Context, id uuid.UUID, customDomain *string) error {
	return nil
}
func (m *mockStoreRepo) DeleteCustomDomain(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockStoreRepo) ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error) {
	return []*model.Store{m.store}, 1, nil
}

func TestTenantMiddleware_ValidTenant(t *testing.T) {
	cd := "shop.acme.test"
	repo := &mockStoreRepo{
		store: &model.Store{
			ID:           uuid.New(),
			Name:         "Acme Test",
			Subdomain:    "acme",
			CustomDomain: &cd,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	mw := middleware.TenantMiddleware(repo, "yourplatform.test", []string{"localhost"})

	var capturedTenant *model.Store
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant, _ = middleware.TenantFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "acme.yourplatform.test:8080"
	rr := httptest.NewRecorder()

	mw(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if capturedTenant == nil || capturedTenant.Name != "Acme Test" {
		t.Fatalf("expected tenant 'Acme Test', got %+v", capturedTenant)
	}
}

func TestTenantMiddleware_NotFoundTenant(t *testing.T) {
	repo := &mockStoreRepo{store: nil}
	mw := middleware.TenantMiddleware(repo, "yourplatform.test", []string{"localhost"})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "unknown.yourplatform.test"
	rr := httptest.NewRecorder()

	mw(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", rr.Code)
	}
}

func TestTenantMiddleware_SystemHostBypass(t *testing.T) {
	repo := &mockStoreRepo{store: nil}
	mw := middleware.TenantMiddleware(repo, "yourplatform.test", []string{"localhost", "127.0.0.1"})

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores", nil)
	req.Host = "localhost:8080"
	rr := httptest.NewRecorder()

	mw(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for system host, got %d", rr.Code)
	}
	if !handlerCalled {
		t.Fatal("expected test handler to be called for system host")
	}
}
