package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/metrics"
	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
)

// DomainHandler handles domain validation requests for Caddy's On-Demand TLS and tenant domain verification.
type DomainHandler struct {
	repo       repository.TenantRepository
	cfg        *config.Config
	systemHost map[string]bool
}

// NewDomainHandler creates a new DomainHandler instance.
func NewDomainHandler(repo repository.TenantRepository, cfg *config.Config) *DomainHandler {
	systemMap := make(map[string]bool)
	for _, h := range cfg.AllowedSystemHosts {
		systemMap[strings.ToLower(strings.TrimSpace(h))] = true
	}

	return &DomainHandler{
		repo:       repo,
		cfg:        cfg,
		systemHost: systemMap,
	}
}

// ValidateDomain handles Caddy's On-Demand TLS verification request.
// Format: GET /api/v1/domains/validate?domain={domain}
// Caddy expects HTTP 200 OK to issue/renew an SSL certificate.
// Non-200 status codes (400, 404, 403) deny certificate issuance.
func (h *DomainHandler) ValidateDomain(w http.ResponseWriter, r *http.Request) {
	rawDomain := r.URL.Query().Get("domain")
	if strings.TrimSpace(rawDomain) == "" {
		log.Printf("[tls-validate] Rejected: empty domain parameter from %s", r.RemoteAddr)
		metrics.DomainValidationsTotal.WithLabelValues("denied", "empty_domain").Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "denied",
			"error":   "Missing required query parameter: domain",
			"allowed": false,
		})
		return
	}

	domain := model.CleanDomain(rawDomain)

	// Check if this domain is an allowed system/platform host
	if h.systemHost[domain] {
		log.Printf("[tls-validate] Approved system domain '%s' for TLS certificate", domain)
		metrics.DomainValidationsTotal.WithLabelValues("allowed", "system_host").Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model.DomainValidationResult{
			Status:  "allowed",
			Domain:  domain,
			Allowed: true,
			Reason:  "system_host",
		})
		return
	}

	// Resolve tenant workspace for this domain or subdomain
	tenant, err := h.repo.ResolveTenantByHost(r.Context(), domain, h.cfg.BaseDomain)
	if err != nil {
		log.Printf("[tls-validate] Denied: domain '%s' does not match any registered tenant workspace", domain)
		metrics.DomainValidationsTotal.WithLabelValues("denied", "unknown_tenant").Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(model.DomainValidationResult{
			Status:  "denied",
			Domain:  domain,
			Allowed: false,
			Reason:  "Domain is not associated with any registered tenant workspace.",
		})
		return
	}

	// Verify tenant is active
	if !tenant.IsActive {
		log.Printf("[tls-validate] Denied: tenant '%s' for domain '%s' is inactive", tenant.ID, domain)
		metrics.DomainValidationsTotal.WithLabelValues("denied", "tenant_inactive").Inc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(model.DomainValidationResult{
			Status:   "denied",
			Domain:   domain,
			Allowed:  false,
			TenantID: &tenant.ID,
			StoreID:  &tenant.ID,
			Reason:   "The tenant workspace associated with this domain is currently inactive.",
		})
		return
	}

	// If this is a custom domain, ensure it is verified before approving TLS certificate issuance
	isPlatformSubdomain := strings.HasSuffix(domain, "."+strings.ToLower(h.cfg.BaseDomain))
	if !isPlatformSubdomain {
		if tenant.DomainStatus != model.DomainStatusVerified &&
			tenant.DomainStatus != model.DomainStatusActive &&
			tenant.DomainStatus != model.DomainStatusProvisioning {
			log.Printf("[tls-validate] Denied: custom domain '%s' has unverified status '%s'", domain, tenant.DomainStatus)
			metrics.DomainValidationsTotal.WithLabelValues("denied", "domain_unverified").Inc()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(model.DomainValidationResult{
				Status:   "denied",
				Domain:   domain,
				Allowed:  false,
				TenantID: &tenant.ID,
				StoreID:  &tenant.ID,
				Reason:   "Custom domain has not completed DNS ownership verification.",
			})
			return
		}
	}

	log.Printf("[tls-validate] Approved tenant domain '%s' for workspace '%s' (%s)", domain, tenant.Name, tenant.ID)
	metrics.DomainValidationsTotal.WithLabelValues("allowed", "tenant_active").Inc()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(model.DomainValidationResult{
		Status:   "allowed",
		Domain:   domain,
		Allowed:  true,
		TenantID: &tenant.ID,
		StoreID:  &tenant.ID,
		Reason:   "Tenant domain verified successfully.",
	})
}
