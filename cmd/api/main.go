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
	storeRepo := repository.NewPostgresStoreRepository(db)
	dnsVerifier := service.NewDNSVerifier()

	authHandler := handler.NewAuthHandler(userRepo, cfg)
	domainHandler := handler.NewDomainHandler(storeRepo, cfg)
	storeHandler := handler.NewStoreHandler(storeRepo, cfg, redisClient, dnsVerifier)
	storefrontHandler := handler.NewStorefrontHandler(cfg)

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

		// Tenant Store Routes
		api.Route("/stores", func(stores chi.Router) {
			// Public store lookup by ID
			stores.Get("/{id}", storeHandler.GetStore)

			// Authenticated Store & Domain Management Routes
			stores.Group(func(protected chi.Router) {
				protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
				protected.Get("/", storeHandler.ListStores)
				protected.Post("/", storeHandler.CreateStore)
				protected.Post("/{id}/custom-domain", storeHandler.ConnectCustomDomain)
				protected.Post("/{id}/verify-domain", storeHandler.VerifyCustomDomain)
				protected.Delete("/{id}/custom-domain", storeHandler.DisconnectCustomDomain)
			})
		})
	})

	// Catch-all route: Dispatches between System Root Index and Tenant Storefronts
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
					"service":     "multi-tenant-saas-platform",
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
						"list_stores":           "GET /api/v1/stores",
						"create_store":          "POST /api/v1/stores",
						"connect_custom_domain": "POST /api/v1/stores/{id}/custom-domain",
						"verify_domain":         "POST /api/v1/stores/{id}/verify-domain",
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

		// 2. Resolve tenant store (First check Redis cache, fallback to PostgreSQL)
		var store *model.Store
		if cached, found := redisClient.GetCachedStore(r.Context(), cleanedHost); found {
			store = cached
		} else {
			dbStore, err := storeRepo.ResolveStoreByHost(r.Context(), cleanedHost, cfg.BaseDomain)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "Tenant store not found",
					"host":    cleanedHost,
					"message": fmt.Sprintf("No active store registered for domain '%s'", cleanedHost),
				})
				return
			}
			store = dbStore
			// Cache resolved store for 10 minutes
			redisClient.SetCachedStore(r.Context(), cleanedHost, store, 10*time.Minute)
		}

		if !store.IsActive {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":    "Store inactive",
				"store_id": store.ID,
				"message":  "This store has been temporarily disabled.",
			})
			return
		}

		// Inject tenant store into request context and serve storefront
		ctx := middleware.WithTenant(r.Context(), store)
		storefrontHandler.ServeStorefront(w, r.WithContext(ctx))
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
