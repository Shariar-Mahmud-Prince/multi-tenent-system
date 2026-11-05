package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
)

// StoreHandler handles store management HTTP endpoints.
type StoreHandler struct {
	repo repository.StoreRepository
	cfg  *config.Config
}

// NewStoreHandler creates a new StoreHandler instance.
func NewStoreHandler(repo repository.StoreRepository, cfg *config.Config) *StoreHandler {
	return &StoreHandler{
		repo: repo,
		cfg:  cfg,
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

	store := model.Store{
		Name:         req.Name,
		Subdomain:    req.Subdomain,
		CustomDomain: req.CustomDomain,
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

	respondJSON(w, http.StatusCreated, store.ToResponse(h.cfg.BaseDomain))
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

	respondJSON(w, http.StatusOK, store.ToResponse(h.cfg.BaseDomain))
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

	stores, total, err := h.repo.ListStores(r.Context(), limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list stores: " + err.Error()})
		return
	}

	var responseList []model.StoreResponse
	for _, s := range stores {
		responseList = append(responseList, s.ToResponse(h.cfg.BaseDomain))
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

	if err := h.repo.UpdateCustomDomain(r.Context(), id, &req.CustomDomain); err != nil {
		if errors.Is(err, repository.ErrStoreNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Store not found"})
			return
		}
		if errors.Is(err, repository.ErrCustomDomainTaken) {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "Custom domain is already connected to another store"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to connect custom domain: " + err.Error()})
		return
	}

	// Fetch updated store
	updated, err := h.repo.GetStoreByID(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message":       "Custom domain connected successfully",
			"custom_domain": req.CustomDomain,
		})
		return
	}

	respondJSON(w, http.StatusOK, updated.ToResponse(h.cfg.BaseDomain))
}

// DisconnectCustomDomain handles DELETE /api/v1/stores/{id}/custom-domain
func (h *StoreHandler) DisconnectCustomDomain(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid store UUID format"})
		return
	}

	if err := h.repo.DeleteCustomDomain(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrStoreNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "Store not found"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to disconnect custom domain: " + err.Error()})
		return
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
