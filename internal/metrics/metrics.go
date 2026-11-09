package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts all handled HTTP requests
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed by method, path pattern, and status code.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks latency distribution across routes
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"method", "path"},
	)

	// ActiveTenants tracks currently active stores
	ActiveTenants = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "saas_active_tenants_total",
			Help: "Current total count of active stores in the platform.",
		},
	)

	// DomainValidationsTotal tracks Caddy ask endpoint validation checks
	DomainValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "saas_domain_validations_total",
			Help: "Count of domain verification queries from Caddy On-Demand TLS.",
		},
		[]string{"status", "reason"},
	)
)

// PrometheusMiddleware captures latency and status code metrics for every HTTP route.
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(ww.Status())
		path := r.URL.Path

		// Group generic dynamic IDs to avoid metric cardinality explosion
		if len(path) > 30 && (path[:15] == "/api/v1/stores/") {
			path = "/api/v1/stores/:id"
		}

		HTTPRequestsTotal.WithLabelValues(r.Method, path, statusCode).Inc()
		HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}
