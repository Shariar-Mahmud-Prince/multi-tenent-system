package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	// Block hostile scanners and automated attack tools unconditionally
	vulnerabilityScannerRegex = regexp.MustCompile(`(?i)(sqlmap|nikto|masscan|zgrab|acunetix|dirbuster|nmap|censys|shodan|wpscan)`)
	// Developer tools and generic HTTP clients blocked in strict mode
	devScraperRegex = regexp.MustCompile(`(?i)(python-requests|curl\/|wget|go-http-client)`)
	// Friendly search engine bots we allow but monitor
	searchEngineRegex = regexp.MustCompile(`(?i)(googlebot|bingbot|yandex|duckduckbot|slurp|baiduspider)`)
)

// BotControlMiddleware filters out aggressive scrapers and malicious automated scanners.
func BotControlMiddleware(strict bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ua := strings.TrimSpace(r.UserAgent())

			// 1. Block empty User-Agent on public storefront routes
			if ua == "" && !strings.HasPrefix(r.URL.Path, "/api/v1") && r.URL.Path != "/healthz" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":  "Access Denied",
					"reason": "Missing User-Agent header",
				})
				return
			}

			// 2. Block hostile vulnerability tools unconditionally on storefront routes
			if !strings.HasPrefix(r.URL.Path, "/api/v1") && r.URL.Path != "/healthz" {
				if vulnerabilityScannerRegex.MatchString(ua) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error":  "Access Denied",
						"reason": "Automated security scanning tool detected",
					})
					return
				}

				// 3. In strict mode, also block generic script clients
				if strict && !searchEngineRegex.MatchString(ua) && devScraperRegex.MatchString(ua) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error":  "Access Denied",
						"reason": "Generic automated script client blocked in strict mode",
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPTokenBucket implements a sliding-window token bucket rate limiter in memory.
type ipVisitor struct {
	tokens     float64
	lastSeen   time.Time
	mu         sync.Mutex
}

// IPRateLimiter provides per-IP request rate limiting with automatic background cleanup.
type IPRateLimiter struct {
	visitors sync.Map
	rate     float64 // tokens added per second
	burst    float64 // max tokens (capacity)
}

// NewIPRateLimiter creates a new rate limiter with the given requests/sec and burst capacity.
func NewIPRateLimiter(rate float64, burst float64) *IPRateLimiter {
	limiter := &IPRateLimiter{
		rate:  rate,
		burst: burst,
	}

	// Periodically clean up stale entries every 5 minutes to prevent memory leaks
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			now := time.Now()
			limiter.visitors.Range(func(key, value interface{}) bool {
				v := value.(*ipVisitor)
				v.mu.Lock()
				if now.Sub(v.lastSeen) > 10*time.Minute {
					limiter.visitors.Delete(key)
				}
				v.mu.Unlock()
				return true
			})
		}
	}()

	return limiter
}

func (limiter *IPRateLimiter) allow(ip string) bool {
	val, _ := limiter.visitors.LoadOrStore(ip, &ipVisitor{
		tokens:   limiter.burst,
		lastSeen: time.Now(),
	})
	v := val.(*ipVisitor)

	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(v.lastSeen).Seconds()
	v.lastSeen = now

	// Replenish tokens based on elapsed time
	v.tokens += elapsed * limiter.rate
	if v.tokens > limiter.burst {
		v.tokens = limiter.burst
	}

	if v.tokens >= 1.0 {
		v.tokens -= 1.0
		return true
	}

	return false
}

// RateLimitMiddleware returns an HTTP middleware that rate-limits by client IP.
func RateLimitMiddleware(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			// Check forward header if behind proxy
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				ip = strings.TrimSpace(parts[0])
			}

			if !limiter.allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":       "Too Many Requests",
					"message":     "Request limit exceeded. Please back off and retry.",
					"retry_after": "1 second",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
