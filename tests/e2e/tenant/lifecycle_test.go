package tenant_test

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

func TestE2E_FullCustomerTenantLifecycle(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 5 * time.Second}

	// Verify API availability
	health, err := client.Get(baseURL + "/healthz")
	if err != nil || health.StatusCode != http.StatusOK {
		t.Skip("Live API is unavailable; skipping E2E lifecycle test")
	}
	health.Body.Close()

	suffix := uuid.New().String()[:8]
	email := fmt.Sprintf("tenant-owner-%s@e2e.test", suffix)
	password := "ProdReadyPass123!"

	// Step 1: Register merchant/workspace owner account
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

	// Step 2: Create a tenant workspace with unique subdomain
	subdomain := fmt.Sprintf("workspace-%s", suffix)
	tenantBody, _ := json.Marshal(model.CreateTenantRequest{
		Name:      "E2E SaaS Workspace",
		Subdomain: subdomain,
	})
	reqTenant, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/tenants", bytes.NewReader(tenantBody))
	reqTenant.Header.Set("Content-Type", "application/json")
	reqTenant.Header.Set("Authorization", "Bearer "+token)

	tenantResp, err := client.Do(reqTenant)
	if err != nil || tenantResp.StatusCode != http.StatusCreated {
		t.Fatalf("Step 2 failed: tenant creation returned status %d", tenantResp.StatusCode)
	}
	var createdTenant model.TenantResponse
	_ = json.NewDecoder(tenantResp.Body).Decode(&createdTenant)
	tenantResp.Body.Close()

	if createdTenant.Subdomain != subdomain {
		t.Fatalf("Step 2 failed: expected subdomain %s, got %s", subdomain, createdTenant.Subdomain)
	}

	// Step 3: Connect a custom domain to the tenant
	customDomain := fmt.Sprintf("app-%s.local", suffix)
	connectBody, _ := json.Marshal(model.ConnectCustomDomainRequest{
		CustomDomain: customDomain,
	})
	reqConnect, _ := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/tenants/%s/custom-domain", baseURL, createdTenant.ID.String()),
		bytes.NewReader(connectBody),
	)
	reqConnect.Header.Set("Content-Type", "application/json")
	reqConnect.Header.Set("Authorization", "Bearer "+token)

	connectResp, err := client.Do(reqConnect)
	if err != nil || connectResp.StatusCode != http.StatusOK {
		t.Fatalf("Step 3 failed: connect custom domain returned status %d", connectResp.StatusCode)
	}
	var connectResult model.TenantResponse
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
		fmt.Sprintf("%s/api/v1/tenants/%s/verify-domain", baseURL, createdTenant.ID.String()),
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

	// Step 5: Resolve tenant application workspace using custom domain Host header
	reqApp, _ := http.NewRequest(http.MethodGet, baseURL+"/", nil)
	reqApp.Host = customDomain
	reqApp.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) SaaS-Tester")

	appResp, err := client.Do(reqApp)
	if err != nil {
		t.Fatalf("Step 5 failed: tenant app request error: %v", err)
	}
	bodyContent, _ := io.ReadAll(appResp.Body)
	appResp.Body.Close()

	if appResp.StatusCode != http.StatusOK {
		t.Fatalf("Step 5 failed: tenant app returned status %d, body: %s", appResp.StatusCode, string(bodyContent))
	}

	// Should contain tenant details
	if !strings.Contains(string(bodyContent), "E2E SaaS Workspace") && !strings.Contains(string(bodyContent), createdTenant.ID.String()) {
		t.Errorf("Step 5 warning: response does not mention tenant name or ID: %s", string(bodyContent))
	}

	// Step 6: Disconnect custom domain
	reqDisconnect, _ := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/api/v1/tenants/%s/custom-domain", baseURL, createdTenant.ID.String()),
		nil,
	)
	reqDisconnect.Header.Set("Authorization", "Bearer "+token)

	disconnectResp, err := client.Do(reqDisconnect)
	if err != nil || disconnectResp.StatusCode != http.StatusOK {
		t.Fatalf("Step 6 failed: disconnect custom domain returned status %d", disconnectResp.StatusCode)
	}
	disconnectResp.Body.Close()
}
