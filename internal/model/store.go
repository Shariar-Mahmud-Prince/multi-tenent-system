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

const (
	DomainStatusPending      = "pending_verification"
	DomainStatusVerified     = "verified"
	DomainStatusProvisioning = "provisioning"
	DomainStatusActive       = "active"
	DomainStatusFailed       = "failed"
	DomainStatusSuspended    = "suspended"
)

var (
	subdomainRegex   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)
	domainRegex      = regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
	reservedSubnames = map[string]bool{
		"admin":      true,
		"api":        true,
		"app":        true,
		"caddy":      true,
		"dev":        true,
		"grafana":    true,
		"internal":   true,
		"mail":       true,
		"monitor":    true,
		"prometheus": true,
		"root":       true,
		"ssl":        true,
		"staging":    true,
		"status":     true,
		"system":     true,
		"test":       true,
		"tls":        true,
		"www":        true,
	}
)

// Store represents a tenant store in the database.
type Store struct {
	ID                    uuid.UUID  `json:"id"`
	OwnerID               *uuid.UUID `json:"owner_id,omitempty"`
	Name                  string     `json:"name"`
	Subdomain             string     `json:"subdomain"`
	CustomDomain          *string    `json:"custom_domain,omitempty"`
	DomainStatus          string     `json:"domain_status"`
	VerificationToken     *string    `json:"verification_token,omitempty"`
	VerificationExpiresAt *time.Time `json:"verification_expires_at,omitempty"`
	IsActive              bool       `json:"is_active"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// FullSubdomain returns the complete subdomain URL host e.g. "acme.yourplatform.local".
func (s *Store) FullSubdomain(baseDomain string) string {
	return fmt.Sprintf("%s.%s", strings.ToLower(s.Subdomain), strings.ToLower(baseDomain))
}

// DNSInstructions outlines how the store owner configures DNS at their registrar.
type DNSInstructions struct {
	CustomDomain string `json:"custom_domain"`
	TXTRecord    struct {
		Host  string `json:"host"`
		Value string `json:"value"`
	} `json:"txt_record"`
	CNAMERecord struct {
		Host   string `json:"host"`
		Target string `json:"target"`
	} `json:"cname_record"`
	ARecord struct {
		Host string `json:"host"`
		IPv4 string `json:"ipv4"`
	} `json:"a_record"`
}

// GenerateDNSInstructions builds DNS setup instructions for a custom domain.
func (s *Store) GenerateDNSInstructions(baseDomain, serverIP string) DNSInstructions {
	var instr DNSInstructions
	if s.CustomDomain == nil || s.VerificationToken == nil {
		return instr
	}

	cd := *s.CustomDomain
	token := *s.VerificationToken

	instr.CustomDomain = cd
	instr.TXTRecord.Host = "_saas-verify." + cd
	instr.TXTRecord.Value = "token=" + token

	instr.CNAMERecord.Host = cd
	instr.CNAMERecord.Target = s.FullSubdomain(baseDomain)

	instr.ARecord.Host = cd
	instr.ARecord.IPv4 = serverIP

	return instr
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
	ID                    uuid.UUID        `json:"id"`
	OwnerID               *uuid.UUID       `json:"owner_id,omitempty"`
	Name                  string           `json:"name"`
	Subdomain             string           `json:"subdomain"`
	SubdomainURL          string           `json:"subdomain_url"`
	CustomDomain          *string          `json:"custom_domain,omitempty"`
	DomainStatus          string           `json:"domain_status"`
	VerificationToken     *string          `json:"verification_token,omitempty"`
	VerificationExpiresAt *time.Time       `json:"verification_expires_at,omitempty"`
	DNSInstructions       *DNSInstructions `json:"dns_instructions,omitempty"`
	IsActive              bool             `json:"is_active"`
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`
}

// ToResponse converts a Store model to a StoreResponse DTO.
func (s *Store) ToResponse(baseDomain string, serverIP string) StoreResponse {
	var dnsInstr *DNSInstructions
	if s.CustomDomain != nil && s.DomainStatus == DomainStatusPending {
		d := s.GenerateDNSInstructions(baseDomain, serverIP)
		dnsInstr = &d
	}

	return StoreResponse{
		ID:                    s.ID,
		OwnerID:               s.OwnerID,
		Name:                  s.Name,
		Subdomain:             s.Subdomain,
		SubdomainURL:          fmt.Sprintf("https://%s.%s", s.Subdomain, baseDomain),
		CustomDomain:          s.CustomDomain,
		DomainStatus:          s.DomainStatus,
		VerificationToken:     s.VerificationToken,
		VerificationExpiresAt: s.VerificationExpiresAt,
		DNSInstructions:       dnsInstr,
		IsActive:              s.IsActive,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
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
