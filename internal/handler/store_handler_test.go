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

func setupStoreHandlerRouter(repo *mockStoreRepository, cfg *config.Config) http.Handler {
	sh := handler.NewStoreHandler(repo, cfg)
	r := chi.NewRouter()

	r.Route("/api/v1/stores", func(stores chi.Router) {
		stores.Post("/", sh.CreateStore)
		stores.Get("/", sh.ListStores)
		stores.Get("/{id}", sh.GetStore)
		stores.Post("/{id}/custom-domain", sh.ConnectCustomDomain)
		stores.Delete("/{id}/custom-domain", sh.DisconnectCustomDomain)
	})

	return r
}

func TestStoreHandler_CreateStore(t *testing.T) {
	cfg := &config.Config{BaseDomain: "yourplatform.test"}
	repo := &mockStoreRepository{}
	r := setupStoreHandlerRouter(repo, cfg)

	payload := model.CreateStoreRequest{
		Name:      "Nike Official",
		Subdomain: "nike",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var res model.StoreResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Subdomain != "nike" || res.SubdomainURL != "https://nike.yourplatform.test" {
		t.Errorf("unexpected store response: %+v", res)
	}
}

func TestStoreHandler_ConnectCustomDomain(t *testing.T) {
	cfg := &config.Config{BaseDomain: "yourplatform.test"}
	storeID := uuid.New()
	repo := &mockStoreRepository{
		stores: []*model.Store{
			{
				ID:        storeID,
				Name:      "Adidas",
				Subdomain: "adidas",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	r := setupStoreHandlerRouter(repo, cfg)

	payload := model.ConnectCustomDomainRequest{
		CustomDomain: "shop.adidas.com",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+storeID.String()+"/custom-domain", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify in mock repository
	store, _ := repo.GetStoreByID(context.Background(), storeID)
	if store.CustomDomain == nil || *store.CustomDomain != "shop.adidas.com" {
		t.Errorf("expected custom domain to be shop.adidas.com, got %v", store.CustomDomain)
	}
}
