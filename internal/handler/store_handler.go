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

// StoreHandler handles store management HTTP endpoints.
type StoreHandler struct {
	repo        repository.StoreRepository
	cfg         *config.Config
	redisClient *database.RedisClient
	dnsVerifier service.DNSVerifier
}

// NewStoreHandler creates a new StoreHandler instance.
func NewStoreHandler(
	repo repository.StoreRepository,
	cfg *config.Config,
	redisClient *database.RedisClient,
	dnsVerifier service.DNSVerifier,
) *StoreHandler {
	return &StoreHandler{
		repo:        repo,
		cfg:         cfg,
		redisClient: redisClient,
		dnsVerifier: dnsVerifier,
	}
}

// CreateStore handles POST /api/v1/stores
func (h *StoreHandler) CreateStore(w http.ResponseWriter, r *http.Request) {
	var req model.CreateStoreRequest
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

	store := model.Store{
		OwnerID:      ownerID,
		Name:         req.Name,
		Subdomain:    req.Subdomain,
		CustomDomain: req.CustomDomain,
		DomainStatus: model.DomainStatusActive,
		IsActive:     true,
	}

	if err := h.repo.CreateStore(r.Context(), &store); err != nil {
		if errors.Is(err, repository.ErrSubdomainTaken) {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "Subdomain is already registered to another store"})
			return
		}
		if errors.Is(err, repository.ErrCustomDomainTaken) {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "Custom domain is already registered to another store"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create store: " + err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, store.ToResponse(h.cfg.BaseDomain, h.cfg.ServerIP))
}

// GetStore handles GET /api/v1/stores/{id}
func (h *StoreHandler) GetStore(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid store UUID format"})
		return
	}

	store, err := h.repo.GetStoreByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrStoreNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Store not found"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve store: " + err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, store.ToResponse(h.cfg.BaseDomain, h.cfg.ServerIP))
}

// ListStores handles GET /api/v1/stores
func (h *StoreHandler) ListStores(w http.ResponseWriter, r *http.Request) {
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

	// If authenticated user is an owner, list only their stores to prevent cross-tenant enumeration
	if claims, ok := middleware.UserFromContext(r.Context()); ok && claims != nil && claims.Role != "admin" {
		stores, err := h.repo.ListStoresByOwner(r.Context(), claims.UserID)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list tenant stores"})
			return
		}
		var responseList []model.StoreResponse
		for _, s := range stores {
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

	stores, total, err := h.repo.ListStores(r.Context(), limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list stores: " + err.Error()})
		return
	}

	var responseList []model.StoreResponse
	for _, s := range stores {
		responseList = append(responseList, s.ToResponse(h.cfg.BaseDomain, h.cfg.ServerIP))
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   responseList,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ConnectCustomDomain handles POST /api/v1/stores/{id}/custom-domain
func (h *StoreHandler) ConnectCustomDomain(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid store UUID format"})
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
		if errors.Is(err, repository.ErrUnauthorizedStore) {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "You do not have permission to manage this store"})
			return
		}
		if errors.Is(err, repository.ErrCustomDomainTaken) {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "Custom domain is already registered to another store"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to register custom domain: " + err.Error()})
		return
	}

	updated, err := h.repo.GetStoreByID(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Custom domain registered. Please configure DNS verification records.",
		})
		return
	}

	// Invalidate cache if redis is configured
	if h.redisClient != nil {
		h.redisClient.InvalidateStoreCache(r.Context(), h.cfg.BaseDomain, updated)
	}

	respondJSON(w, http.StatusOK, updated.ToResponse(h.cfg.BaseDomain, h.cfg.ServerIP))
}

// VerifyCustomDomain handles POST /api/v1/stores/{id}/verify-domain
func (h *StoreHandler) VerifyCustomDomain(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid store UUID format"})
		return
	}

	store, err := h.repo.GetStoreByID(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Store not found"})
		return
	}

	// IDOR check
	if claims, ok := middleware.UserFromContext(r.Context()); ok && claims != nil && claims.Role != "admin" {
		if store.OwnerID == nil || *store.OwnerID != claims.UserID {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "You do not have permission to verify this store"})
			return
		}
	}

	if store.CustomDomain == nil || *store.CustomDomain == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "No custom domain is connected to this store"})
		return
	}

	if store.VerificationToken == nil || *store.VerificationToken == "" {
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
			*store.CustomDomain,
			*store.VerificationToken,
			store.FullSubdomain(h.cfg.BaseDomain),
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

	store.DomainStatus = model.DomainStatusActive
	if h.redisClient != nil {
		h.redisClient.InvalidateStoreCache(r.Context(), h.cfg.BaseDomain, store)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"verified":      true,
		"status":        model.DomainStatusActive,
		"custom_domain": *store.CustomDomain,
		"message":       "Domain ownership verified successfully! SSL certificate is ready to be provisioned.",
	})
}

// DisconnectCustomDomain handles DELETE /api/v1/stores/{id}/custom-domain
func (h *StoreHandler) DisconnectCustomDomain(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid store UUID format"})
		return
	}

	var ownerID *uuid.UUID
	if claims, ok := middleware.UserFromContext(r.Context()); ok && claims != nil && claims.Role != "admin" {
		ownerID = &claims.UserID
	}

	store, _ := h.repo.GetStoreByID(r.Context(), id)

	if err := h.repo.DeleteCustomDomain(r.Context(), id, ownerID); err != nil {
		if errors.Is(err, repository.ErrUnauthorizedStore) {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "You do not have permission to manage this store"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to disconnect custom domain: " + err.Error()})
		return
	}

	if store != nil && h.redisClient != nil {
		h.redisClient.InvalidateStoreCache(r.Context(), h.cfg.BaseDomain, store)
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
