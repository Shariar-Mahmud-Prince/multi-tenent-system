package model

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	subdomainRegex   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)
	domainRegex      = regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
	reservedSubnames = map[string]bool{
		"admin":    true,
		"api":      true,
		"app":      true,
		"caddy":    true,
		"dev":      true,
		"internal": true,
		"mail":     true,
		"root":     true,
		"ssl":      true,
		"staging":  true,
		"status":   true,
		"system":   true,
		"test":     true,
		"tls":      true,
		"www":      true,
	}
)

// Store represents a tenant store in the database.
type Store struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Subdomain    string    `json:"subdomain"`
	CustomDomain *string   `json:"custom_domain,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// FullSubdomain returns the complete subdomain URL host e.g. "acme.yourplatform.test".
func (s *Store) FullSubdomain(baseDomain string) string {
	return fmt.Sprintf("%s.%s", strings.ToLower(s.Subdomain), strings.ToLower(baseDomain))
}

// CreateStoreRequest contains the payload for creating a new store.
type CreateStoreRequest struct {
	Name         string  `json:"name"`
	Subdomain    string  `json:"subdomain"`
	CustomDomain *string `json:"custom_domain,omitempty"`
}

// Validate validates the store creation request.
func (r *CreateStoreRequest) Validate(baseDomain string) error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("store name is required")
	}
	if len(r.Name) > 255 {
		return errors.New("store name must not exceed 255 characters")
	}

	r.Subdomain = strings.ToLower(strings.TrimSpace(r.Subdomain))
	if r.Subdomain == "" {
		return errors.New("subdomain is required")
	}
	if len(r.Subdomain) < 3 || len(r.Subdomain) > 63 {
		return errors.New("subdomain must be between 3 and 63 characters")
	}
	if !subdomainRegex.MatchString(r.Subdomain) {
		return errors.New("subdomain must contain only lowercase alphanumeric characters and hyphens, and cannot start or end with a hyphen")
	}
	if reservedSubnames[r.Subdomain] {
		return fmt.Errorf("subdomain '%s' is a reserved platform keyword", r.Subdomain)
	}

	if r.CustomDomain != nil {
		cd := CleanDomain(*r.CustomDomain)
		if cd == "" {
			r.CustomDomain = nil
		} else {
			if err := ValidateCustomDomain(cd, baseDomain); err != nil {
				return err
			}
			r.CustomDomain = &cd
		}
	}

	return nil
}

// ConnectCustomDomainRequest contains payload for connecting or updating a custom domain.
type ConnectCustomDomainRequest struct {
	CustomDomain string `json:"custom_domain"`
}

// Validate validates the custom domain input.
func (r *ConnectCustomDomainRequest) Validate(baseDomain string) error {
	r.CustomDomain = CleanDomain(r.CustomDomain)
	if r.CustomDomain == "" {
		return errors.New("custom_domain is required")
	}
	return ValidateCustomDomain(r.CustomDomain, baseDomain)
}

// CleanDomain normalizes a domain string by trimming whitespace, lowercasing, and stripping port or protocol.
func CleanDomain(raw string) string {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.ToLower(cleaned)

	// Strip http:// or https:// if provided
	if strings.HasPrefix(cleaned, "http://") {
		cleaned = strings.TrimPrefix(cleaned, "http://")
	} else if strings.HasPrefix(cleaned, "https://") {
		cleaned = strings.TrimPrefix(cleaned, "https://")
	}

	// Remove trailing slash and path
	if idx := strings.Index(cleaned, "/"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	// Strip port if present (e.g. localhost:8080 or domain.com:443)
	if host, _, err := net.SplitHostPort(cleaned); err == nil {
		cleaned = host
	}

	// Remove trailing dot if present
	cleaned = strings.TrimSuffix(cleaned, ".")

	return cleaned
}

// ValidateCustomDomain validates that a custom domain is well-formed and not an internal platform domain.
func ValidateCustomDomain(domain string, baseDomain string) error {
	if domain == "" {
		return errors.New("domain cannot be empty")
	}
	if len(domain) > 255 {
		return errors.New("domain must not exceed 255 characters")
	}
	if !domainRegex.MatchString(domain) {
		return errors.New("domain must be a valid FQDN (e.g. shop.example.com or mystore.com)")
	}

	baseDomain = strings.ToLower(strings.TrimSpace(baseDomain))
	if domain == baseDomain || strings.HasSuffix(domain, "."+baseDomain) {
		return fmt.Errorf("custom domain cannot be a platform subdomain of %s", baseDomain)
	}

	return nil
}

// StoreResponse is the serialized representation returned to clients.
type StoreResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Subdomain    string    `json:"subdomain"`
	SubdomainURL string    `json:"subdomain_url"`
	CustomDomain *string   `json:"custom_domain,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToResponse converts a Store model to a StoreResponse DTO.
func (s *Store) ToResponse(baseDomain string) StoreResponse {
	return StoreResponse{
		ID:           s.ID,
		Name:         s.Name,
		Subdomain:    s.Subdomain,
		SubdomainURL: fmt.Sprintf("https://%s.%s", s.Subdomain, baseDomain),
		CustomDomain: s.CustomDomain,
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// DomainValidationResult represents the response for Caddy's ask endpoint.
type DomainValidationResult struct {
	Status  string     `json:"status"`
	Domain  string     `json:"domain"`
	Allowed bool       `json:"allowed"`
	StoreID *uuid.UUID `json:"store_id,omitempty"`
	Reason  string     `json:"reason,omitempty"`
}
