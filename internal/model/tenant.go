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

// Tenant represents an isolated multi-tenant application/workspace instance in the SaaS platform.
type Tenant struct {
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

// Backward compatibility alias
type Store = Tenant

// FullSubdomain returns the complete subdomain URL host e.g. "acme.yourplatform.local".
func (t *Tenant) FullSubdomain(baseDomain string) string {
	return fmt.Sprintf("%s.%s", strings.ToLower(t.Subdomain), strings.ToLower(baseDomain))
}

// DNSInstructions outlines how the tenant owner configures DNS at their registrar.
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
func (t *Tenant) GenerateDNSInstructions(baseDomain, serverIP string) DNSInstructions {
	var instr DNSInstructions
	if t.CustomDomain == nil || t.VerificationToken == nil {
		return instr
	}

	cd := *t.CustomDomain
	token := *t.VerificationToken

	instr.CustomDomain = cd
	instr.TXTRecord.Host = "_saas-verify." + cd
	instr.TXTRecord.Value = "token=" + token

	instr.CNAMERecord.Host = cd
	instr.CNAMERecord.Target = t.FullSubdomain(baseDomain)

	instr.ARecord.Host = cd
	instr.ARecord.IPv4 = serverIP

	return instr
}

// CreateTenantRequest contains the payload for provisioning a new tenant workspace.
type CreateTenantRequest struct {
	Name         string  `json:"name"`
	Subdomain    string  `json:"subdomain"`
	CustomDomain *string `json:"custom_domain,omitempty"`
}

// Backward compatibility alias
type CreateStoreRequest = CreateTenantRequest

// Validate validates the tenant creation request.
func (r *CreateTenantRequest) Validate(baseDomain string) error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("tenant name is required")
	}
	if len(r.Name) > 255 {
		return errors.New("tenant name must not exceed 255 characters")
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
		return errors.New("domain must be a valid FQDN (e.g. app.customer.com or workspace.domain.org)")
	}

	baseDomain = strings.ToLower(strings.TrimSpace(baseDomain))
	if domain == baseDomain || strings.HasSuffix(domain, "."+baseDomain) {
		return fmt.Errorf("custom domain cannot be a platform subdomain of %s", baseDomain)
	}

	return nil
}

// TenantResponse is the serialized representation returned to clients.
type TenantResponse struct {
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

// Backward compatibility alias
type StoreResponse = TenantResponse

// ToResponse converts a Tenant model to a TenantResponse DTO.
func (t *Tenant) ToResponse(baseDomain string, serverIP string) TenantResponse {
	var dnsInstr *DNSInstructions
	if t.CustomDomain != nil && t.DomainStatus == DomainStatusPending {
		d := t.GenerateDNSInstructions(baseDomain, serverIP)
		dnsInstr = &d
	}

	return TenantResponse{
		ID:                    t.ID,
		OwnerID:               t.OwnerID,
		Name:                  t.Name,
		Subdomain:             t.Subdomain,
		SubdomainURL:          fmt.Sprintf("https://%s.%s", t.Subdomain, baseDomain),
		CustomDomain:          t.CustomDomain,
		DomainStatus:          t.DomainStatus,
		VerificationToken:     t.VerificationToken,
		VerificationExpiresAt: t.VerificationExpiresAt,
		DNSInstructions:       dnsInstr,
		IsActive:              t.IsActive,
		CreatedAt:             t.CreatedAt,
		UpdatedAt:             t.UpdatedAt,
	}
}

// DomainValidationResult represents the response for Caddy's ask endpoint.
type DomainValidationResult struct {
	Status   string     `json:"status"`
	Domain   string     `json:"domain"`
	Allowed  bool       `json:"allowed"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
	StoreID  *uuid.UUID `json:"store_id,omitempty"` // Backward compatibility
	Reason   string     `json:"reason,omitempty"`
}
