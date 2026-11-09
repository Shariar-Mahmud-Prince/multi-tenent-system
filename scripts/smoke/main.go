package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	fmt.Println("=================================================================")
	fmt.Println("   🧪 MULTI-TENANT DEPLOYMENT & FEATURE SMOKE TEST")
	fmt.Println("=================================================================")

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	passCount := 0
	failCount := 0

	check := func(name string, success bool, details string) {
		if success {
			fmt.Printf(" [PASS] %s\n        ↳ %s\n", name, details)
			passCount++
		} else {
			fmt.Printf(" [FAIL] %s\n        ↳ %s\n", name, details)
			failCount++
		}
	}

	// 1. Healthcheck
	resp, err := client.Get("http://localhost:8080/healthz")
	check("Backend Healthcheck", err == nil && resp != nil && resp.StatusCode == http.StatusOK, "HTTP GET /healthz returned 200 OK")
	if resp != nil {
		resp.Body.Close()
	}

	// 2. Caddy TLS Validation: Known Tenant Domain
	resp, err = client.Get("http://localhost:8080/api/v1/domains/validate?domain=ecom.local")
	check("Caddy TLS Validation (Active Domain)", err == nil && resp != nil && resp.StatusCode == http.StatusOK, "Domain 'ecom.local' returned 200 OK (Allowed)")
	if resp != nil {
		resp.Body.Close()
	}

	// 3. Caddy TLS Validation: Unauthorized Attacker Domain
	resp, err = client.Get("http://localhost:8080/api/v1/domains/validate?domain=malicious-attacker.com")
	check("Caddy TLS Validation (Unknown Domain Blocked)", err == nil && resp != nil && resp.StatusCode == http.StatusNotFound, "Domain 'malicious-attacker.com' returned 404 (Denied)")
	if resp != nil {
		resp.Body.Close()
	}

	// 4. Storefront Routing via Caddy Port 80
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:80/", nil)
	req.Host = "ecom.local"
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	resp, err = client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	check("Storefront Multi-Tenant Resolution", err == nil && resp.StatusCode == http.StatusOK && bytes.Contains(body, []byte("Acme Store")), "Host 'ecom.local' resolved to 'Acme Store'")

	// 5. Bot Traffic Control: Block Empty User-Agent
	req, _ = http.NewRequest(http.MethodGet, "http://localhost:8080/storefront", nil)
	req.Host = "ecom.local"
	req.Header.Set("User-Agent", "")
	resp, err = client.Do(req)
	check("Bot Traffic Filter (Missing User-Agent)", err == nil && resp != nil && resp.StatusCode == http.StatusForbidden, "Empty User-Agent blocked with 403 Forbidden")
	if resp != nil {
		resp.Body.Close()
	}

	// 6. Store Creation REST API
	storePayload := map[string]interface{}{
		"name":      fmt.Sprintf("Smoke Store %d", time.Now().Unix()),
		"subdomain": fmt.Sprintf("smoke-%d", time.Now().Unix()%10000),
	}
	payloadBytes, _ := json.Marshal(storePayload)
	resp, err = client.Post("http://localhost:8080/api/v1/stores", "application/json", bytes.NewReader(payloadBytes))
	check("Store Management API (Create Store)", err == nil && resp != nil && resp.StatusCode == http.StatusCreated, fmt.Sprintf("Created store subdomain '%s'", storePayload["subdomain"]))
	if resp != nil {
		resp.Body.Close()
	}

	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf(" SUMMARY: %d Passed, %d Failed\n", passCount, failCount)
	fmt.Println("=================================================================")
}
