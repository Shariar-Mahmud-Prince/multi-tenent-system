package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/handler"
	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
)

// mockStoreRepository is an in-memory mock implementing repository.StoreRepository
type mockStoreRepository struct {
	stores []*model.Store
}

func (m *mockStoreRepository) CreateStore(ctx context.Context, store *model.Store) error {
	store.ID = uuid.New()
	store.CreatedAt = time.Now()
	store.UpdatedAt = time.Now()
	m.stores = append(m.stores, store)
	return nil
}

func (m *mockStoreRepository) GetStoreByID(ctx context.Context, id uuid.UUID) (*model.Store, error) {
	for _, s := range m.stores {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, repository.ErrStoreNotFound
}

func (m *mockStoreRepository) GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error) {
	for _, s := range m.stores {
		if s.Subdomain == subdomain {
			return s, nil
		}
	}
	return nil, repository.ErrStoreNotFound
}

func (m *mockStoreRepository) GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error) {
	for _, s := range m.stores {
		if s.CustomDomain != nil && *s.CustomDomain == customDomain {
			return s, nil
		}
	}
	return nil, repository.ErrStoreNotFound
}

func (m *mockStoreRepository) ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error) {
	cleaned := model.CleanDomain(host)
	// Try subdomain
	suffix := "." + baseDomain
	if len(cleaned) > len(suffix) && cleaned[len(cleaned)-len(suffix):] == suffix {
		sub := cleaned[:len(cleaned)-len(suffix)]
		for _, s := range m.stores {
			if s.Subdomain == sub {
				return s, nil
			}
		}
	}
	// Try custom domain
	for _, s := range m.stores {
		if s.CustomDomain != nil && *s.CustomDomain == cleaned {
			return s, nil
		}
	}
	return nil, repository.ErrStoreNotFound
}

func (m *mockStoreRepository) UpdateCustomDomain(ctx context.Context, id uuid.UUID, customDomain *string) error {
	for _, s := range m.stores {
		if s.ID == id {
			s.CustomDomain = customDomain
			return nil
		}
	}
	return repository.ErrStoreNotFound
}

func (m *mockStoreRepository) DeleteCustomDomain(ctx context.Context, id uuid.UUID) error {
	return m.UpdateCustomDomain(ctx, id, nil)
}

func (m *mockStoreRepository) ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error) {
	return m.stores, len(m.stores), nil
}

func setupTestDomainHandler() (*handler.DomainHandler, *mockStoreRepository, *config.Config) {
	cfg := &config.Config{
		BaseDomain:         "yourplatform.test",
		AllowedSystemHosts: []string{"localhost", "yourplatform.test", "api.yourplatform.test"},
	}

	custom := "acmeshop.com"
	inactiveCustom := "inactiveshop.com"
	repo := &mockStoreRepository{
		stores: []*model.Store{
			{
				ID:           uuid.New(),
				Name:         "Acme Store",
				Subdomain:    "acme",
				CustomDomain: &custom,
				IsActive:     true,
			},
			{
				ID:           uuid.New(),
				Name:         "Inactive Store",
				Subdomain:    "inactive",
				CustomDomain: &inactiveCustom,
				IsActive:     false,
			},
		},
	}

	h := handler.NewDomainHandler(repo, cfg)
	return h, repo, cfg
}

func TestValidateDomain_SystemDomain(t *testing.T) {
	h, _, _ := setupTestDomainHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/domains/validate?domain=yourplatform.test", nil)
	rr := httptest.NewRecorder()

	h.ValidateDomain(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for system domain, got %d", rr.Code)
	}

	var res model.DomainValidationResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	if !res.Allowed || res.Reason != "system_host" {
		t.Errorf("unexpected validation result: %+v", res)
	}
}

func TestValidateDomain_TenantSubdomain(t *testing.T) {
	h, _, _ := setupTestDomainHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/domains/validate?domain=acme.yourplatform.test", nil)
	rr := httptest.NewRecorder()

	h.ValidateDomain(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid tenant subdomain, got %d: %s", rr.Code, rr.Body.String())
	}

	var res model.DomainValidationResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	if !res.Allowed || res.StoreID == nil {
		t.Errorf("expected allowed=true with store ID, got: %+v", res)
	}
}

func TestValidateDomain_TenantCustomDomain(t *testing.T) {
	h, _, _ := setupTestDomainHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/domains/validate?domain=acmeshop.com", nil)
	rr := httptest.NewRecorder()

	h.ValidateDomain(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid tenant custom domain, got %d", rr.Code)
	}
}

func TestValidateDomain_InactiveStore(t *testing.T) {
	h, _, _ := setupTestDomainHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/domains/validate?domain=inactive.yourplatform.test", nil)
	rr := httptest.NewRecorder()

	h.ValidateDomain(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for inactive store, got %d", rr.Code)
	}
}

func TestValidateDomain_UnknownDomain(t *testing.T) {
	h, _, _ := setupTestDomainHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/domains/validate?domain=randomattacker.com", nil)
	rr := httptest.NewRecorder()

	h.ValidateDomain(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for unknown domain, got %d", rr.Code)
	}
}

func TestValidateDomain_MissingQueryParam(t *testing.T) {
	h, _, _ := setupTestDomainHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/domains/validate", nil)
	rr := httptest.NewRecorder()

	h.ValidateDomain(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when query param missing, got %d", rr.Code)
	}
}
