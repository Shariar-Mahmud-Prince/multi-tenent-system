package ratelimit_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"multi-tenant-domains/internal/middleware"
)

func TestSecurity_RateLimit_BurstThrottling(t *testing.T) {
	// Limiter: 10 req/s, burst 5
	limiter := middleware.NewIPRateLimiter(10.0, 5.0)
	mw := middleware.RateLimitMiddleware(limiter)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	totalRequests := 20
	var successCount int32
	var throttledCount int32

	var wg sync.WaitGroup
	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "203.0.113.45:54321" // specific attacker IP
			rr := httptest.NewRecorder()

			mw(handler).ServeHTTP(rr, req)

			if rr.Code == http.StatusOK {
				atomic.AddInt32(&successCount, 1)
			} else if rr.Code == http.StatusTooManyRequests {
				atomic.AddInt32(&throttledCount, 1)
				retryAfter := rr.Header().Get("Retry-After")
				if retryAfter == "" {
					t.Errorf("missing Retry-After header on 429 response")
				}
			}
		}()
	}
	wg.Wait()

	if throttledCount == 0 {
		t.Fatalf("RATE LIMIT FAILURE! No requests were throttled despite sending %d rapid concurrent requests", totalRequests)
	}

	if successCount > 6 {
		t.Errorf("expected at most burst capacity of 5-6 allowed requests, got %d", successCount)
	}
}

func TestSecurity_BotScannerBlocking(t *testing.T) {
	mw := middleware.BotControlMiddleware(false)
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	maliciousUserAgents := []string{
		"sqlmap/1.5#stable",
		"Nikto/2.1.6",
		"wpscan v3.8.20",
		"masscan/1.0.5",
		"Nmap Scripting Engine",
		"DirBuster-1.0-RC1",
	}

	for _, ua := range maliciousUserAgents {
		t.Run("Scanner: "+ua, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req.Header.Set("User-Agent", ua)
			rr := httptest.NewRecorder()

			mw(dummyHandler).ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Errorf("BOT CONTROL FAILURE: User-Agent %q was not blocked, got status %d", ua, rr.Code)
			}
		})
	}
}
