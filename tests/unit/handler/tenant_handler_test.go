package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/handler"
	"multi-tenant-domains/internal/model"
)

func setupTenantHandlerRouter(repo *mockStoreRepository, cfg *config.Config) http.Handler {
	th := handler.NewTenantHandler(repo, cfg, nil, nil)
	r := chi.NewRouter()

	r.Route("/api/v1/tenants", func(tenants chi.Router) {
		tenants.Post("/", th.CreateTenant)
		tenants.Get("/", th.ListTenants)
		tenants.Get("/{id}", th.GetTenant)
		tenants.Post("/{id}/custom-domain", th.ConnectCustomDomain)
		tenants.Delete("/{id}/custom-domain", th.DisconnectCustomDomain)
	})

	// Legacy alias routes
	r.Route("/api/v1/stores", func(stores chi.Router) {
		stores.Post("/", th.CreateStore)
		stores.Get("/", th.ListStores)
		stores.Get("/{id}", th.GetStore)
		stores.Post("/{id}/custom-domain", th.ConnectCustomDomain)
		stores.Delete("/{id}/custom-domain", th.DisconnectCustomDomain)
	})

	return r
}

func TestTenantHandler_CreateTenant(t *testing.T) {
	cfg := &config.Config{BaseDomain: "yourplatform.test"}
	repo := &mockStoreRepository{}
	r := setupTenantHandlerRouter(repo, cfg)

	payload := model.CreateTenantRequest{
		Name:      "Acme Workspace",
		Subdomain: "acme-workspace",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var res model.TenantResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Subdomain != "acme-workspace" || res.SubdomainURL != "https://acme-workspace.yourplatform.test" {
		t.Errorf("unexpected tenant response: %+v", res)
	}
}

func TestTenantHandler_ConnectCustomDomain(t *testing.T) {
	cfg := &config.Config{BaseDomain: "yourplatform.test"}
	tenantID := uuid.New()
	repo := &mockStoreRepository{
		stores: []*model.Tenant{
			{
				ID:        tenantID,
				Name:      "Enterprise Portal",
				Subdomain: "enterprise",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	r := setupTenantHandlerRouter(repo, cfg)

	payload := model.ConnectCustomDomainRequest{
		CustomDomain: "app.enterprise.com",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+tenantID.String()+"/custom-domain", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify in mock repository
	tenant, _ := repo.GetTenantByID(context.Background(), tenantID)
	if tenant.CustomDomain == nil || *tenant.CustomDomain != "app.enterprise.com" {
		t.Errorf("expected custom domain to be app.enterprise.com, got %v", tenant.CustomDomain)
	}
}
