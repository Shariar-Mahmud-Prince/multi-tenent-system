package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/database"
	"multi-tenant-domains/internal/middleware"
	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
	"multi-tenant-domains/internal/service"
)

// TenantHandler handles tenant workspace management HTTP endpoints.
type TenantHandler struct {
	repo        repository.TenantRepository
	cfg         *config.Config
	redisClient *database.RedisClient
	dnsVerifier service.DNSVerifier
}

// Backward compatibility alias
type StoreHandler = TenantHandler

// NewTenantHandler creates a new TenantHandler instance.
func NewTenantHandler(
	repo repository.TenantRepository,
	cfg *config.Config,
	redisClient *database.RedisClient,
	dnsVerifier service.DNSVerifier,
) *TenantHandler {
	return &TenantHandler{
		repo:        repo,
		cfg:         cfg,
		redisClient: redisClient,
		dnsVerifier: dnsVerifier,
	}
}

// NewStoreHandler is a backward-compatible constructor.
func NewStoreHandler(
	repo repository.StoreRepository,
	cfg *config.Config,
	redisClient *database.RedisClient,
	dnsVerifier service.DNSVerifier,
) *StoreHandler {
	return NewTenantHandler(repo, cfg, redisClient, dnsVerifier)
}

// CreateTenant handles POST /api/v1/tenants
func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON payload: " + err.Error()})
		return
	}

	if err := req.Validate(h.cfg.BaseDomain); err != nil {
		respondJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	var ownerID *uuid.UUID
	if claims, ok := middleware.UserFromContext(r.Context()); ok && claims != nil {
		ownerID = &claims.UserID
	}

	tenant := model.Tenant{
		OwnerID:      ownerID,
		Name:         req.Name,
		Subdomain:    req.Subdomain,
		CustomDomain: req.CustomDomain,
		DomainStatus: model.DomainStatusActive,
		IsActive:     true,
	}

	if err := h.repo.CreateTenant(r.Context(), &tenant); err != nil {
		if errors.Is(err, repository.ErrSubdomainTaken) {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "Subdomain is already registered to another tenant"})
			return
		}
		if errors.Is(err, repository.ErrCustomDomainTaken) {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "Custom domain is already registered to another tenant"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create tenant: " + err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, tenant.ToResponse(h.cfg.BaseDomain, h.cfg.ServerIP))
}

func (h *TenantHandler) CreateStore(w http.ResponseWriter, r *http.Request) {
	h.CreateTenant(w, r)
}

// GetTenant handles GET /api/v1/tenants/{id}
func (h *TenantHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant UUID format"})
		return
	}

	tenant, err := h.repo.GetTenantByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Tenant not found"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve tenant: " + err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, tenant.ToResponse(h.cfg.BaseDomain, h.cfg.ServerIP))
}

func (h *TenantHandler) GetStore(w http.ResponseWriter, r *http.Request) {
	h.GetTenant(w, r)
}

// ListTenants handles GET /api/v1/tenants
func (h *TenantHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	// If authenticated user is an owner, list only their tenants to prevent cross-tenant enumeration
	if claims, ok := middleware.UserFromContext(r.Context()); ok && claims != nil && claims.Role != "admin" {
		tenants, err := h.repo.ListTenantsByOwner(r.Context(), claims.UserID)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list tenant workspaces"})
			return
		}
		var responseList []model.TenantResponse
		for _, s := range tenants {
			responseList = append(responseList, s.ToResponse(h.cfg.BaseDomain, h.cfg.ServerIP))
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"data":   responseList,
			"total":  len(responseList),
			"limit":  limit,
			"offset": 0,
		})
		return
	}

	tenants, total, err := h.repo.ListTenants(r.Context(), limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list tenants: " + err.Error()})
		return
	}

	var responseList []model.TenantResponse
	for _, s := range tenants {
		responseList = append(responseList, s.ToResponse(h.cfg.BaseDomain, h.cfg.ServerIP))
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   responseList,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *TenantHandler) ListStores(w http.ResponseWriter, r *http.Request) {
	h.ListTenants(w, r)
}

// ConnectCustomDomain handles POST /api/v1/tenants/{id}/custom-domain
func (h *TenantHandler) ConnectCustomDomain(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant UUID format"})
		return
	}

	var req model.ConnectCustomDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON payload: " + err.Error()})
		return
	}

	if err := req.Validate(h.cfg.BaseDomain); err != nil {
		respondJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	// Verify owner ID if user is authenticated
	var ownerID *uuid.UUID
	if claims, ok := middleware.UserFromContext(r.Context()); ok && claims != nil && claims.Role != "admin" {
		ownerID = &claims.UserID
	}

	// Generate 32-byte (64 hex characters) secure verification token
	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	verificationToken := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(48 * time.Hour)

	if err := h.repo.UpdateCustomDomainWithVerification(
		r.Context(),
		id,
		ownerID,
		req.CustomDomain,
		verificationToken,
		expiresAt,
	); err != nil {
		if errors.Is(err, repository.ErrUnauthorizedTenant) {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "You do not have permission to manage this tenant"})
			return
		}
		if errors.Is(err, repository.ErrCustomDomainTaken) {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "Custom domain is already registered to another tenant"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to register custom domain: " + err.Error()})
		return
	}

	updated, err := h.repo.GetTenantByID(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Custom domain registered. Please configure DNS verification records.",
		})
		return
	}

	// Invalidate cache if redis is configured
	if h.redisClient != nil {
		h.redisClient.InvalidateTenantCache(r.Context(), h.cfg.BaseDomain, updated)
	}

	respondJSON(w, http.StatusOK, updated.ToResponse(h.cfg.BaseDomain, h.cfg.ServerIP))
}

// VerifyCustomDomain handles POST /api/v1/tenants/{id}/verify-domain
func (h *TenantHandler) VerifyCustomDomain(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant UUID format"})
		return
	}

	tenant, err := h.repo.GetTenantByID(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Tenant not found"})
		return
	}

	// IDOR check
	if claims, ok := middleware.UserFromContext(r.Context()); ok && claims != nil && claims.Role != "admin" {
		if tenant.OwnerID == nil || *tenant.OwnerID != claims.UserID {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "You do not have permission to verify this tenant"})
			return
		}
	}

	if tenant.CustomDomain == nil || *tenant.CustomDomain == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "No custom domain is connected to this tenant"})
		return
	}

	if tenant.VerificationToken == nil || *tenant.VerificationToken == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "No pending verification token exists for this domain"})
		return
	}

	// Perform DNS verification
	var verified bool
	var reason string
	if h.dnsVerifier != nil {
		var verifyErr error
		verified, reason, verifyErr = h.dnsVerifier.VerifyOwnership(
			r.Context(),
			*tenant.CustomDomain,
			*tenant.VerificationToken,
			tenant.FullSubdomain(h.cfg.BaseDomain),
		)
		err = verifyErr
	} else {
		verified = true
	}

	if !verified {
		_ = h.repo.UpdateDomainStatus(r.Context(), id, model.DomainStatusFailed)
		respondJSON(w, http.StatusExpectationFailed, map[string]interface{}{
			"verified": false,
			"status":   model.DomainStatusFailed,
			"reason":   reason,
			"details":  err,
		})
		return
	}

	// Verified! Transition to active
	if err := h.repo.UpdateDomainStatus(r.Context(), id, model.DomainStatusActive); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update domain status"})
		return
	}

	tenant.DomainStatus = model.DomainStatusActive
	if h.redisClient != nil {
		h.redisClient.InvalidateTenantCache(r.Context(), h.cfg.BaseDomain, tenant)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"verified":      true,
		"status":        model.DomainStatusActive,
		"custom_domain": *tenant.CustomDomain,
		"message":       "Domain ownership verified successfully! SSL certificate is ready to be provisioned.",
	})
}

// DisconnectCustomDomain handles DELETE /api/v1/tenants/{id}/custom-domain
func (h *TenantHandler) DisconnectCustomDomain(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tenant UUID format"})
		return
	}

	var ownerID *uuid.UUID
	if claims, ok := middleware.UserFromContext(r.Context()); ok && claims != nil && claims.Role != "admin" {
		ownerID = &claims.UserID
	}

	tenant, _ := h.repo.GetTenantByID(r.Context(), id)

	if err := h.repo.DeleteCustomDomain(r.Context(), id, ownerID); err != nil {
		if errors.Is(err, repository.ErrUnauthorizedTenant) {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "You do not have permission to manage this tenant"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to disconnect custom domain: " + err.Error()})
		return
	}

	if tenant != nil && h.redisClient != nil {
		h.redisClient.InvalidateTenantCache(r.Context(), h.cfg.BaseDomain, tenant)
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Custom domain disconnected successfully",
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
