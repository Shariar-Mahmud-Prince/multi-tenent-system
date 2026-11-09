package ssl_test

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCaddy_HTTPToHTTPSRedirect(t *testing.T) {
	// Custom transport that does not follow redirects
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:80/", nil)
	req.Host = "ecom.local"

	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("Caddy port 80 not reachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently && resp.StatusCode != http.StatusPermanentRedirect {
		t.Fatalf("expected 301 or 308 redirect from HTTP to HTTPS, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if !strings.HasPrefix(location, "https://") {
		t.Errorf("expected Location header to begin with https://, got %q", location)
	}
}

func TestCaddy_SecurityHeadersAndTLS(t *testing.T) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Internal Caddy CA certificate
			ServerName:         "ecom.local",
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	req, _ := http.NewRequest(http.MethodGet, "https://127.0.0.1:443/", nil)
	req.Host = "ecom.local"
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) SaaS-Tester")

	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("Caddy port 443 not reachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK via HTTPS, got %d", resp.StatusCode)
	}

	// Verify HSTS
	hsts := resp.Header.Get("Strict-Transport-Security")
	if !strings.Contains(hsts, "max-age=") {
		t.Errorf("missing or invalid Strict-Transport-Security header: %q", hsts)
	}

	// Verify X-Content-Type-Options
	nosniff := resp.Header.Get("X-Content-Type-Options")
	if nosniff != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %q", nosniff)
	}

	// Verify X-Frame-Options
	frameOptions := resp.Header.Get("X-Frame-Options")
	if frameOptions == "" {
		t.Errorf("missing X-Frame-Options header")
	}

	// Verify Server header or Via
	via := resp.Header.Get("Via")
	if !strings.Contains(via, "Caddy") {
		t.Logf("Via header: %q", via)
	}
}

func TestCaddy_OnDemandTLSAskHook(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Authorized platform domain
	respPlatform, err := client.Get("http://127.0.0.1:8080/api/v1/domains/validate?domain=ecom.local")
	if err != nil {
		t.Skipf("API server not reachable: %v", err)
	}
	defer respPlatform.Body.Close()

	if respPlatform.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for registered store ecom.local, got %d", respPlatform.StatusCode)
	}

	// 2. Unauthorized attacker domain
	respAttacker, err := client.Get("http://127.0.0.1:8080/api/v1/domains/validate?domain=malicious-phishing-host.com")
	if err != nil {
		t.Fatalf("API request failed: %v", err)
	}
	defer respAttacker.Body.Close()

	if respAttacker.StatusCode != http.StatusNotFound && respAttacker.StatusCode != http.StatusForbidden {
		t.Errorf("expected 404 or 403 for unauthorized domain, got %d", respAttacker.StatusCode)
	}
}
