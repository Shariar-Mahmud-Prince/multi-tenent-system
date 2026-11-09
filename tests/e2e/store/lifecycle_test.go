package store_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"multi-tenant-domains/internal/model"
)

func getBaseURL() string {
	url := os.Getenv("API_URL")
	if url == "" {
		url = "http://127.0.0.1:8080"
	}
	return url
}

func TestE2E_FullCustomerStoreLifecycle(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 5 * time.Second}

	// Verify API availability
	health, err := client.Get(baseURL + "/healthz")
	if err != nil || health.StatusCode != http.StatusOK {
		t.Skip("Live API is unavailable; skipping E2E lifecycle test")
	}
	health.Body.Close()

	suffix := uuid.New().String()[:8]
	email := fmt.Sprintf("merchant-%s@e2e.test", suffix)
	password := "ProdReadyPass123!"

	// Step 1: Register merchant account
	regBody, _ := json.Marshal(model.RegisterUserRequest{
		Email:    email,
		Password: password,
		Role:     "owner",
	})
	regResp, err := client.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	if err != nil || regResp.StatusCode != http.StatusCreated {
		t.Fatalf("Step 1 failed: user registration returned status %d", regResp.StatusCode)
	}
	var authResp model.AuthResponse
	_ = json.NewDecoder(regResp.Body).Decode(&authResp)
	regResp.Body.Close()

	token := authResp.Token
	if token == "" {
		t.Fatalf("Step 1 failed: empty JWT received")
	}

	// Step 2: Create a store with unique subdomain
	subdomain := fmt.Sprintf("e2estore-%s", suffix)
	storeBody, _ := json.Marshal(model.CreateStoreRequest{
		Name:      "E2E Test Boutique",
		Subdomain: subdomain,
	})
	reqStore, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/stores", bytes.NewReader(storeBody))
	reqStore.Header.Set("Content-Type", "application/json")
	reqStore.Header.Set("Authorization", "Bearer "+token)

	storeResp, err := client.Do(reqStore)
	if err != nil || storeResp.StatusCode != http.StatusCreated {
		t.Fatalf("Step 2 failed: store creation returned status %d", storeResp.StatusCode)
	}
	var createdStore model.StoreResponse
	_ = json.NewDecoder(storeResp.Body).Decode(&createdStore)
	storeResp.Body.Close()

	if createdStore.Subdomain != subdomain {
		t.Fatalf("Step 2 failed: expected subdomain %s, got %s", subdomain, createdStore.Subdomain)
	}

	// Step 3: Connect a custom domain
	customDomain := fmt.Sprintf("shop-%s.local", suffix)
	connectBody, _ := json.Marshal(model.ConnectCustomDomainRequest{
		CustomDomain: customDomain,
	})
	reqConnect, _ := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/stores/%s/custom-domain", baseURL, createdStore.ID.String()),
		bytes.NewReader(connectBody),
	)
	reqConnect.Header.Set("Content-Type", "application/json")
	reqConnect.Header.Set("Authorization", "Bearer "+token)

	connectResp, err := client.Do(reqConnect)
	if err != nil || connectResp.StatusCode != http.StatusOK {
		t.Fatalf("Step 3 failed: connect custom domain returned status %d", connectResp.StatusCode)
	}
	var connectResult model.StoreResponse
	_ = json.NewDecoder(connectResp.Body).Decode(&connectResult)
	connectResp.Body.Close()

	if connectResult.DomainStatus != model.DomainStatusPending {
		t.Errorf("Step 3 warning: expected domain status %s, got %s", model.DomainStatusPending, connectResult.DomainStatus)
	}
	if connectResult.VerificationToken == nil || *connectResult.VerificationToken == "" {
		t.Fatalf("Step 3 failed: missing verification token")
	}

	// Step 4: Verify custom domain ownership
	reqVerify, _ := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/stores/%s/verify-domain", baseURL, createdStore.ID.String()),
		nil,
	)
	reqVerify.Header.Set("Authorization", "Bearer "+token)

	verifyResp, err := client.Do(reqVerify)
	if err != nil || verifyResp.StatusCode != http.StatusOK {
		t.Fatalf("Step 4 failed: verify domain returned status %d", verifyResp.StatusCode)
	}
	var verifyResult map[string]interface{}
	_ = json.NewDecoder(verifyResp.Body).Decode(&verifyResult)
	verifyResp.Body.Close()

	if verified, ok := verifyResult["verified"].(bool); !ok || !verified {
		t.Fatalf("Step 4 failed: domain verification reported false: %+v", verifyResult)
	}

	// Step 5: Resolve storefront using the verified custom domain Host header
	reqStorefront, _ := http.NewRequest(http.MethodGet, baseURL+"/", nil)
	reqStorefront.Host = customDomain
	reqStorefront.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) SaaS-Tester")

	sfResp, err := client.Do(reqStorefront)
	if err != nil {
		t.Fatalf("Step 5 failed: storefront request error: %v", err)
	}
	bodyContent, _ := io.ReadAll(sfResp.Body)
	sfResp.Body.Close()

	if sfResp.StatusCode != http.StatusOK {
		t.Fatalf("Step 5 failed: storefront returned status %d, body: %s", sfResp.StatusCode, string(bodyContent))
	}

	// Should contain store details
	if !strings.Contains(string(bodyContent), "E2E Test Boutique") && !strings.Contains(string(bodyContent), createdStore.ID.String()) {
		t.Errorf("Step 5 warning: storefront response does not mention store name or ID: %s", string(bodyContent))
	}

	// Step 6: Disconnect custom domain
	reqDisconnect, _ := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/api/v1/stores/%s/custom-domain", baseURL, createdStore.ID.String()),
		nil,
	)
	reqDisconnect.Header.Set("Authorization", "Bearer "+token)

	disconnectResp, err := client.Do(reqDisconnect)
	if err != nil || disconnectResp.StatusCode != http.StatusOK {
		t.Fatalf("Step 6 failed: disconnect custom domain returned status %d", disconnectResp.StatusCode)
	}
	disconnectResp.Body.Close()
}
