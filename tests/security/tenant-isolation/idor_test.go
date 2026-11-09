package tenantisolation_test

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

func registerAndLogin(t *testing.T, email, password string) string {
	baseURL := getBaseURL()
	payload := model.RegisterUserRequest{
		Email:    email,
		Password: password,
		Role:     "owner",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		var authResp model.AuthResponse
		_ = json.NewDecoder(resp.Body).Decode(&authResp)
		return authResp.Token
	}

	// Try login if already registered
	loginPayload := model.LoginRequest{
		Email:    email,
		Password: password,
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginResp, err := http.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("failed to login user: %v", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to login: status %d", loginResp.StatusCode)
	}

	var authResp model.AuthResponse
	_ = json.NewDecoder(loginResp.Body).Decode(&authResp)
	return authResp.Token
}

func createStore(t *testing.T, token, name, subdomain string) model.StoreResponse {
	baseURL := getBaseURL()
	payload := model.CreateStoreRequest{
		Name:      name,
		Subdomain: subdomain,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/stores", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created on store creation, got %d", resp.StatusCode)
	}

	var storeResp model.StoreResponse
	_ = json.NewDecoder(resp.Body).Decode(&storeResp)
	return storeResp
}

func TestIDOR_TenantIsolation_CustomDomainOperations(t *testing.T) {
	baseURL := getBaseURL()
	client := &http.Client{Timeout: 5 * time.Second}

	// Skip if API server is not running
	healthResp, err := client.Get(baseURL + "/healthz")
	if err != nil || healthResp.StatusCode != http.StatusOK {
		t.Skip("Live API server not accessible; skipping live IDOR integration test")
	}
	healthResp.Body.Close()

	// 1. Create User A and Store A
	suffix := uuid.New().String()[:8]
	userAEmail := fmt.Sprintf("tenant-a-%s@security.test", suffix)
	tokenA := registerAndLogin(t, userAEmail, "SecurePassword123!")
	storeA := createStore(t, tokenA, "Tenant A Store", "tenant-a-"+suffix)

	// 2. Create User B and Store B
	userBEmail := fmt.Sprintf("tenant-b-%s@security.test", suffix)
	tokenB := registerAndLogin(t, userBEmail, "SecurePassword123!")
	storeB := createStore(t, tokenB, "Tenant B Store", "tenant-b-"+suffix)

	// 3. Attack 1: User B tries to attach custom domain to Store A (IDOR Attack)
	attackPayload := model.ConnectCustomDomainRequest{
		CustomDomain: fmt.Sprintf("hijack-%s.com", suffix),
	}
	attackBody, _ := json.Marshal(attackPayload)
	reqAttack, _ := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/stores/%s/custom-domain", baseURL, storeA.ID.String()),
		bytes.NewReader(attackBody),
	)
	reqAttack.Header.Set("Content-Type", "application/json")
	reqAttack.Header.Set("Authorization", "Bearer "+tokenB) // Attacker's token

	respAttack, err := client.Do(reqAttack)
	if err != nil {
		t.Fatalf("attack request failed: %v", err)
	}
	defer respAttack.Body.Close()

	if respAttack.StatusCode != http.StatusForbidden {
		t.Fatalf("IDOR VULNERABILITY DETECTED! Expected 403 Forbidden for cross-tenant domain update, got %d", respAttack.StatusCode)
	}

	// 4. Attack 2: User B tries to delete custom domain from Store A
	reqDelete, _ := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/api/v1/stores/%s/custom-domain", baseURL, storeA.ID.String()),
		nil,
	)
	reqDelete.Header.Set("Authorization", "Bearer "+tokenB)

	respDelete, err := client.Do(reqDelete)
	if err != nil {
		t.Fatalf("delete attack request failed: %v", err)
	}
	defer respDelete.Body.Close()

	if respDelete.StatusCode != http.StatusForbidden {
		t.Fatalf("IDOR VULNERABILITY DETECTED! Expected 403 Forbidden for cross-tenant domain delete, got %d", respDelete.StatusCode)
	}

	// 5. Cross-tenant Store Enumeration check: User B lists stores -> should NOT see Store A
	reqList, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/stores", nil)
	reqList.Header.Set("Authorization", "Bearer "+tokenB)

	respList, err := client.Do(reqList)
	if err != nil {
		t.Fatalf("store list request failed: %v", err)
	}
	defer respList.Body.Close()

	var listData struct {
		Data []model.StoreResponse `json:"data"`
	}
	_ = json.NewDecoder(respList.Body).Decode(&listData)

	for _, s := range listData.Data {
		if s.ID == storeA.ID {
			t.Fatalf("CROSS-TENANT LEAKAGE! User B was able to view Store A (%s) in their store list", storeA.ID)
		}
	}

	// Verify User B DOES see Store B
	foundStoreB := false
	for _, s := range listData.Data {
		if s.ID == storeB.ID {
			foundStoreB = true
			break
		}
	}
	if !foundStoreB {
		t.Errorf("User B failed to see their own Store B in store list")
	}
}
