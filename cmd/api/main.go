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

	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/database"
	"multi-tenant-domains/internal/handler"
	"multi-tenant-domains/internal/middleware"
	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
)

var version = "1.0.0"

func main() {
	log.Printf("[init] Starting Multi-Tenant Domain Microservice v%s...", version)

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

	// 3. Execute database migrations
	if err := database.RunMigrations(db, cfg.MigrationsDir); err != nil {
		log.Fatalf("[fatal] Schema migrations failed: %v", err)
	}

	// 4. Instantiate repositories and handlers
	storeRepo := repository.NewPostgresStoreRepository(db)
	domainHandler := handler.NewDomainHandler(storeRepo, cfg)
	storeHandler := handler.NewStoreHandler(storeRepo, cfg)
	storefrontHandler := handler.NewStorefrontHandler(cfg)

	// 5. Build router
	r := chi.NewRouter()

	// Global standard middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))

	// Health check endpoint (for Docker, Caddy, Kubernetes)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   version,
		})
	})

	// Platform API Routes (Caddy ask validation & store management)
	r.Route("/api/v1", func(api chi.Router) {
		// Caddy On-Demand TLS Verification Hook
		api.Get("/domains/validate", domainHandler.ValidateDomain)

		// Tenant Store Management
		api.Route("/stores", func(stores chi.Router) {
			stores.Post("/", storeHandler.CreateStore)
			stores.Get("/", storeHandler.ListStores)
			stores.Get("/{id}", storeHandler.GetStore)
			stores.Post("/{id}/custom-domain", storeHandler.ConnectCustomDomain)
			stores.Delete("/{id}/custom-domain", storeHandler.DisconnectCustomDomain)
		})
	})

	// Catch-all route: Dispatches between System Root Index and Tenant Storefronts
	systemHostsMap := make(map[string]bool)
	for _, h := range cfg.AllowedSystemHosts {
		systemHostsMap[strings.ToLower(strings.TrimSpace(h))] = true
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		cleanedHost := model.CleanDomain(r.Host)

		// If accessed via system/platform host and requesting the root/docs
		if systemHostsMap[cleanedHost] {
			if r.URL.Path == "/" || r.URL.Path == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"service":     "multi-tenant-domain-service",
					"version":     version,
					"base_domain": cfg.BaseDomain,
					"status":      "running",
					"endpoints": map[string]string{
						"health":                "GET /healthz",
						"caddy_tls_validate":    "GET /api/v1/domains/validate?domain={domain}",
						"create_store":          "POST /api/v1/stores",
						"list_stores":           "GET /api/v1/stores",
						"get_store":             "GET /api/v1/stores/{id}",
						"connect_custom_domain": "POST /api/v1/stores/{id}/custom-domain",
					},
				})
				return
			}

			// If unknown endpoint on system host
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "API route not found on platform host",
				"path":  r.URL.Path,
			})
			return
		}

		// Otherwise, resolve tenant store from host header
		store, err := storeRepo.ResolveStoreByHost(r.Context(), cleanedHost, cfg.BaseDomain)
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

		// Inactive checks passed, inject tenant into context and serve storefront
		ctx := middleware.WithTenant(r.Context(), store)
		storefrontHandler.ServeStorefront(w, r.WithContext(ctx))
	})

	// 6. Start HTTP Server with Graceful Shutdown
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
