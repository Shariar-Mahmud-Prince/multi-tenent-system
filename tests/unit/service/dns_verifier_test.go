package service_test

import (
	"context"
	"testing"
	"time"

	"multi-tenant-domains/internal/service"
)

func TestDNSVerifier_LocalDomains(t *testing.T) {
	verifier := service.NewDNSVerifier()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// .local domain should pass automatically in dev
	verified, reason, err := verifier.VerifyOwnership(ctx, "mystore.local", "token123", "sub.platform.local")
	if err != nil {
		t.Fatalf("unexpected error for .local domain: %v", err)
	}
	if !verified {
		t.Fatalf("expected .local domain to be auto-verified, got reason: %s", reason)
	}

	// .test domain should pass automatically in dev
	verifiedTest, reasonTest, err := verifier.VerifyOwnership(ctx, "acme.test", "tokenXYZ", "acme.platform.test")
	if err != nil {
		t.Fatalf("unexpected error for .test domain: %v", err)
	}
	if !verifiedTest {
		t.Fatalf("expected .test domain to be auto-verified, got reason: %s", reasonTest)
	}
}

func TestDNSVerifier_UnresolvablePublicDomain(t *testing.T) {
	verifier := service.NewDNSVerifier()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Non-existent domain should fail verification
	verified, reason, _ := verifier.VerifyOwnership(ctx, "nonexistent-random-domain-xyz98734.com", "randomtoken", "target.saas.com")
	if verified {
		t.Fatalf("expected unresolvable public domain to fail verification, but passed with reason: %s", reason)
	}
}
