package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/database"
	"multi-tenant-domains/internal/handler"
	"multi-tenant-domains/internal/metrics"
	"multi-tenant-domains/internal/middleware"
	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
	"multi-tenant-domains/internal/service"
)

var version = "2.0.0"

func main() {
	log.Printf("[init] Starting Production-Grade Multi-Tenant SaaS Platform v%s...", version)

	// 1. Load configuration
	cfg := config.LoadConfig()
	log.Printf("[init] Environment: %s | Base Domain: %s | Port: %s", cfg.Env, cfg.BaseDomain, cfg.ServerPort)

	// 2. Connect to PostgreSQL
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("[fatal] Database initialization failed: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("[error] Failed to close database: %v", err)
		}
	}()

	// 3. Connect to Redis (graceful degradation if offline)
	redisClient := database.NewRedisClient(cfg)
	defer func() {
		if redisClient != nil {
			_ = redisClient.Close()
		}
	}()

	// 4. Execute database migrations
	if err := database.RunMigrations(db, cfg.MigrationsDir); err != nil {
		log.Fatalf("[fatal] Schema migrations failed: %v", err)
	}

	// 5. Instantiate services, repositories, and handlers
	userRepo := repository.NewPostgresUserRepository(db)
	tenantRepo := repository.NewPostgresTenantRepository(db)
	dnsVerifier := service.NewDNSVerifier()

	authHandler := handler.NewAuthHandler(userRepo, cfg)
	domainHandler := handler.NewDomainHandler(tenantRepo, cfg)
	tenantHandler := handler.NewTenantHandler(tenantRepo, cfg, redisClient, dnsVerifier)
	tenantAppHandler := handler.NewTenantAppHandler(cfg)

	// 6. Build Chi router
	r := chi.NewRouter()

	// Global production middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))
	r.Use(metrics.PrometheusMiddleware)

	// Bot Traffic Control & Dynamic IP Rate Limiting (50 req/sec, burst 100)
	rateLimiter := middleware.NewIPRateLimiter(50.0, 100.0)
	r.Use(middleware.RateLimitMiddleware(rateLimiter))
	r.Use(middleware.BotControlMiddleware(false))

	// Health check endpoint (for Docker, Caddy, Kubernetes)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   version,
			"database":  "connected",
			"redis":     redisClient != nil,
		})
	})

	// Prometheus Metrics Endpoint (internal monitoring)
	r.Handle("/metrics", promhttp.Handler())

	// Platform API Routes
	r.Route("/api/v1", func(api chi.Router) {
		// Public Authentication Routes
		api.Route("/auth", func(auth chi.Router) {
			auth.Post("/register", authHandler.Register)
			auth.Post("/login", authHandler.Login)

			// Authenticated Profile Route
			auth.Group(func(protected chi.Router) {
				protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
				protected.Get("/me", authHandler.Me)
			})
		})

		// Caddy On-Demand TLS Verification Hook (Public)
		api.Get("/domains/validate", domainHandler.ValidateDomain)

		// Primary Multi-Tenant Workspace Routes
		registerTenantRoutes := func(router chi.Router) {
			// Public tenant lookup by ID
			router.Get("/{id}", tenantHandler.GetTenant)

			// Authenticated Tenant & Domain Management Routes
			router.Group(func(protected chi.Router) {
				protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
				protected.Get("/", tenantHandler.ListTenants)
				protected.Post("/", tenantHandler.CreateTenant)
				protected.Post("/{id}/custom-domain", tenantHandler.ConnectCustomDomain)
				protected.Post("/{id}/verify-domain", tenantHandler.VerifyCustomDomain)
				protected.Delete("/{id}/custom-domain", tenantHandler.DisconnectCustomDomain)
			})
		}

		api.Route("/tenants", registerTenantRoutes)
		// Backward compatibility alias: /stores routes to /tenants
		api.Route("/stores", registerTenantRoutes)
	})

	// Catch-all route: Dispatches between System Root Index and Tenant Applications
	systemHostsMap := make(map[string]bool)
	for _, h := range cfg.AllowedSystemHosts {
		systemHostsMap[strings.ToLower(strings.TrimSpace(h))] = true
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		cleanedHost := model.CleanDomain(r.Host)

		// 1. If accessed via system/platform host, display platform API index
		if systemHostsMap[cleanedHost] {
			if r.URL.Path == "/" || r.URL.Path == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"service":     "multi-tenant-saas-foundation",
					"version":     version,
					"base_domain": cfg.BaseDomain,
					"status":      "running",
					"endpoints": map[string]string{
						"health":                "GET /healthz",
						"metrics":               "GET /metrics",
						"register":              "POST /api/v1/auth/register",
						"login":                 "POST /api/v1/auth/login",
						"profile":               "GET /api/v1/auth/me",
						"caddy_tls_validate":    "GET /api/v1/domains/validate?domain={domain}",
						"list_tenants":          "GET /api/v1/tenants",
						"create_tenant":         "POST /api/v1/tenants",
						"connect_custom_domain": "POST /api/v1/tenants/{id}/custom-domain",
						"verify_domain":         "POST /api/v1/tenants/{id}/verify-domain",
					},
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "API route not found on platform host",
				"path":  r.URL.Path,
			})
			return
		}

		// 2. Resolve tenant workspace (First check Redis cache, fallback to PostgreSQL)
		var tenant *model.Tenant
		if cached, found := redisClient.GetCachedTenant(r.Context(), cleanedHost); found {
			tenant = cached
		} else {
			dbTenant, err := tenantRepo.ResolveTenantByHost(r.Context(), cleanedHost, cfg.BaseDomain)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "Tenant workspace not found",
					"host":    cleanedHost,
					"message": fmt.Sprintf("No active tenant workspace registered for domain '%s'", cleanedHost),
				})
				return
			}
			tenant = dbTenant
			// Cache resolved tenant for 10 minutes
			redisClient.SetCachedTenant(r.Context(), cleanedHost, tenant, 10*time.Minute)
		}

		if !tenant.IsActive {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":     "Tenant workspace inactive",
				"tenant_id": tenant.ID,
				"message":   "This tenant workspace has been temporarily suspended or disabled.",
			})
			return
		}

		// Inject tenant workspace into request context and serve tenant app
		ctx := middleware.WithTenant(r.Context(), tenant)
		tenantAppHandler.ServeTenantApp(w, r.WithContext(ctx))
	})

	// 7. Start HTTP Server with Graceful Shutdown
	addr := fmt.Sprintf("0.0.0.0:%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[server] HTTP listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Listen for OS interrupt signals
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErrors:
		log.Fatalf("[fatal] Server error: %v", err)

	case sig := <-shutdownSignals:
		log.Printf("[shutdown] Signal received: %v. Initiating graceful shutdown...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[shutdown] Error during graceful shutdown: %v", err)
			_ = srv.Close()
		}
		log.Printf("[shutdown] Server stopped cleanly.")
	}
}
