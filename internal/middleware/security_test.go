package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"multi-tenant-domains/internal/middleware"
)

func TestBotControlMiddleware_EmptyUserAgent(t *testing.T) {
	mw := middleware.BotControlMiddleware(false)
	nextCalled := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Empty UA on storefront route should be blocked
	req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	req.Header.Set("User-Agent", "")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for empty User-Agent, got %d", rr.Code)
	}
	if nextCalled {
		t.Fatal("expected downstream handler NOT to be called")
	}

	// Normal UA should be allowed
	req2 := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid browser UA, got %d", rr2.Code)
	}
}

func TestBotControlMiddleware_MaliciousScannerBlocked(t *testing.T) {
	mw := middleware.BotControlMiddleware(true)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	scanners := []string{
		"sqlmap/1.5.2#stable",
		"Nikto/2.1.6",
		"masscan/1.0",
		"Mozilla/5.0 (compatible; CensysInspect/1.1)",
	}

	for _, scanner := range scanners {
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		req.Header.Set("User-Agent", scanner)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected scanner %q to be blocked with 403, got %d", scanner, rr.Code)
		}
	}
}

func TestIPRateLimiter_BurstAndThrottle(t *testing.T) {
	// 5 requests per second, burst capacity of 3
	limiter := middleware.NewIPRateLimiter(5.0, 3.0)
	mw := middleware.RateLimitMiddleware(limiter)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "198.51.100.42"

	// First 3 requests (burst) must succeed immediately
	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("request %d in burst expected 200 OK, got %d", i, rr.Code)
		}
	}

	// 4th immediate request should be throttled (429 Too Many Requests)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip + ":12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 4th immediate request to return 429 Too Many Requests, got %d", rr.Code)
	}

	// Wait 250ms for token replenishment (at 5 tokens/sec = 1 token every 200ms)
	time.Sleep(250 * time.Millisecond)

	reqAfterWait := httptest.NewRequest(http.MethodGet, "/", nil)
	reqAfterWait.RemoteAddr = ip + ":12345"
	rrAfterWait := httptest.NewRecorder()
	handler.ServeHTTP(rrAfterWait, reqAfterWait)

	if rrAfterWait.Code != http.StatusOK {
		t.Fatalf("expected request after replenishment to succeed with 200 OK, got %d", rrAfterWait.Code)
	}
}
