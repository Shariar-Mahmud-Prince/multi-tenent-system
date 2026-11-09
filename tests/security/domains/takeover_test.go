package domains_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

func registerUser(t *testing.T, email string) string {
	baseURL := getBaseURL()
	payload := model.RegisterUserRequest{
		Email:    email,
		Password: "SecurePassword123!",
		Role:     "owner",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}
	defer resp.Body.Close()

	var authResp model.AuthResponse
	_ = json.NewDecoder(resp.Body).Decode(&authResp)
	return authResp.Token
}

func TestDomainTakeoverPrevention(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 5 * time.Second}

	healthResp, err := client.Get(baseURL + "/healthz")
	if err != nil || healthResp.StatusCode != http.StatusOK {
		t.Skip("Live API server not accessible; skipping domain takeover test")
	}
	healthResp.Body.Close()

	suffix := uuid.New().String()[:8]
	tokenA := registerUser(t, fmt.Sprintf("owner-a-%s@security.test", suffix))
	tokenB := registerUser(t, fmt.Sprintf("owner-b-%s@security.test", suffix))

	// 1. Owner A creates store with subdomain "brand-abc-<suffix>"
	subdomainA := fmt.Sprintf("brand-a-%s", suffix)
	createStoreReq := model.CreateStoreRequest{
		Name:      "Brand A Store",
		Subdomain: subdomainA,
	}
	bodyA, _ := json.Marshal(createStoreReq)
	reqA, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/stores", bytes.NewReader(bodyA))
	reqA.Header.Set("Content-Type", "application/json")
	reqA.Header.Set("Authorization", "Bearer "+tokenA)

	respA, err := client.Do(reqA)
	if err != nil || respA.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create store A: %v", err)
	}
	var storeA model.StoreResponse
	_ = json.NewDecoder(respA.Body).Decode(&storeA)
	respA.Body.Close()

	// 2. Owner B attempts to claim the exact same subdomain -> must return 409 Conflict
	dupReq := model.CreateStoreRequest{
		Name:      "Impostor Store",
		Subdomain: subdomainA,
	}
	dupBody, _ := json.Marshal(dupReq)
	reqB, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/stores", bytes.NewReader(dupBody))
	reqB.Header.Set("Content-Type", "application/json")
	reqB.Header.Set("Authorization", "Bearer "+tokenB)

	respB, err := client.Do(reqB)
	if err != nil {
		t.Fatalf("failed to send duplicate request: %v", err)
	}
	defer respB.Body.Close()

	if respB.StatusCode != http.StatusConflict {
		t.Fatalf("SUBDOMAIN TAKEOVER RISK! Expected 409 Conflict for duplicate subdomain, got %d", respB.StatusCode)
	}

	// 3. Connect a custom domain to Store A in pending_verification status
	customDomain := fmt.Sprintf("custom-%s.example.org", suffix)
	connectReq := model.ConnectCustomDomainRequest{CustomDomain: customDomain}
	connectBody, _ := json.Marshal(connectReq)
	reqConnect, _ := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/stores/%s/custom-domain", baseURL, storeA.ID.String()),
		bytes.NewReader(connectBody),
	)
	reqConnect.Header.Set("Content-Type", "application/json")
	reqConnect.Header.Set("Authorization", "Bearer "+tokenA)

	respConnect, err := client.Do(reqConnect)
	if err != nil || respConnect.StatusCode != http.StatusOK {
		t.Fatalf("failed to connect custom domain: %v", err)
	}
	respConnect.Body.Close()

	// 4. Test Caddy on-demand TLS hook for unverified domain:
	// While domain is pending_verification, Caddy validation MUST return 403 or 404 (NOT 200 OK)
	// This prevents TLS certificate issuance before domain ownership is cryptographically proven!
	validateURL := fmt.Sprintf("%s/api/v1/domains/validate?domain=%s", baseURL, customDomain)
	respValidate, err := client.Get(validateURL)
	if err != nil {
		t.Fatalf("validate request failed: %v", err)
	}
	defer respValidate.Body.Close()

	if respValidate.StatusCode == http.StatusOK {
		t.Fatalf("TLS ON-DEMAND HIJACKING RISK! Caddy approved TLS cert for domain in pending_verification status!")
	}

	// 5. Owner B attempts to claim the same custom domain on Store B -> must return 409 Conflict
	subdomainB := fmt.Sprintf("brand-b-%s", suffix)
	createBReq := model.CreateStoreRequest{
		Name:      "Brand B Store",
		Subdomain: subdomainB,
	}
	bBody, _ := json.Marshal(createBReq)
	reqCreateB, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/stores", bytes.NewReader(bBody))
	reqCreateB.Header.Set("Content-Type", "application/json")
	reqCreateB.Header.Set("Authorization", "Bearer "+tokenB)
	respCreateB, _ := client.Do(reqCreateB)
	var storeB model.StoreResponse
	_ = json.NewDecoder(respCreateB.Body).Decode(&storeB)
	respCreateB.Body.Close()

	dupCustomReq := model.ConnectCustomDomainRequest{CustomDomain: customDomain}
	dupCustomBody, _ := json.Marshal(dupCustomReq)
	reqDupCustom, _ := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/stores/%s/custom-domain", baseURL, storeB.ID.String()),
		bytes.NewReader(dupCustomBody),
	)
	reqDupCustom.Header.Set("Content-Type", "application/json")
	reqDupCustom.Header.Set("Authorization", "Bearer "+tokenB)

	respDupCustom, err := client.Do(reqDupCustom)
	if err != nil {
		t.Fatalf("duplicate custom domain request failed: %v", err)
	}
	defer respDupCustom.Body.Close()

	if respDupCustom.StatusCode != http.StatusConflict {
		t.Fatalf("CUSTOM DOMAIN COLLISION RISK! Expected 409 Conflict, got %d", respDupCustom.StatusCode)
	}
}
