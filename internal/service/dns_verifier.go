package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// DNSVerifier defines the interface for checking DNS records of customer domains.
type DNSVerifier interface {
	VerifyOwnership(ctx context.Context, customDomain string, expectedToken string, targetSubdomain string) (bool, string, error)
}

type realDNSVerifier struct {
	resolver *net.Resolver
}

// NewDNSVerifier instantiates a DNS verification service.
func NewDNSVerifier() DNSVerifier {
	return &realDNSVerifier{
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 3 * time.Second}
				// Default to standard reliable public DNS
				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		},
	}
}

// VerifyOwnership checks TXT record `_saas-verify.<domain>` and/or CNAME record.
func (v *realDNSVerifier) VerifyOwnership(
	ctx context.Context,
	customDomain string,
	expectedToken string,
	targetSubdomain string,
) (bool, string, error) {
	customDomain = strings.ToLower(strings.TrimSpace(customDomain))

	// Special allowance for local test domains during development
	if strings.HasSuffix(customDomain, ".local") || strings.HasSuffix(customDomain, ".test") {
		return true, "Verified automatically for local development environment", nil
	}

	// 1. Check TXT record on _saas-verify.<domain>
	txtHost := fmt.Sprintf("_saas-verify.%s", customDomain)
	txtRecords, err := v.resolver.LookupTXT(ctx, txtHost)
	if err == nil {
		expectedContent := "token=" + expectedToken
		for _, record := range txtRecords {
			if strings.TrimSpace(record) == expectedContent || strings.TrimSpace(record) == expectedToken {
				return true, "TXT record ownership verified successfully", nil
			}
		}
	}

	// 2. Fallback check: CNAME record pointing to target subdomain
	cname, err := v.resolver.LookupCNAME(ctx, customDomain)
	if err == nil {
		cnameClean := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(cname)), ".")
		targetClean := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(targetSubdomain)), ".")
		if cnameClean == targetClean {
			return true, "CNAME record routing verified successfully", nil
		}
	}

	return false, "Neither valid TXT verification record nor matching CNAME record was detected", nil
}
