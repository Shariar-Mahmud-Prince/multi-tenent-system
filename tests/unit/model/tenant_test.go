package model_test

import (
	"testing"

	"multi-tenant-domains/internal/model"
)

func TestCleanDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "example.com"},
		{"  EXAMPLE.COM  ", "example.com"},
		{"http://example.com", "example.com"},
		{"https://shop.acme.org:8443/", "shop.acme.org"},
		{"sub.domain.test:8080", "sub.domain.test"},
		{"store.test.", "store.test"},
		{"HTTPS://MYSTORE.COM/PATH", "mystore.com"},
	}

	for _, tt := range tests {
		actual := model.CleanDomain(tt.input)
		if actual != tt.expected {
			t.Errorf("CleanDomain(%q) = %q; want %q", tt.input, actual, tt.expected)
		}
	}
}

func TestCreateTenantRequest_Validate(t *testing.T) {
	baseDomain := "yourplatform.test"

	tests := []struct {
		name      string
		req       model.CreateTenantRequest
		expectErr bool
	}{
		{
			name: "Valid request",
			req: model.CreateTenantRequest{
				Name:      "Valid Workspace",
				Subdomain: "valid-workspace",
			},
			expectErr: false,
		},
		{
			name: "Empty name",
			req: model.CreateTenantRequest{
				Name:      "",
				Subdomain: "myworkspace",
			},
			expectErr: true,
		},
		{
			name: "Reserved subdomain",
			req: model.CreateTenantRequest{
				Name:      "Admin App",
				Subdomain: "admin",
			},
			expectErr: true,
		},
		{
			name: "Too short subdomain",
			req: model.CreateTenantRequest{
				Name:      "Short App",
				Subdomain: "ab",
			},
			expectErr: true,
		},
		{
			name: "Invalid characters in subdomain",
			req: model.CreateTenantRequest{
				Name:      "Invalid App",
				Subdomain: "my_app!",
			},
			expectErr: true,
		},
		{
			name: "Subdomain starting with hyphen",
			req: model.CreateTenantRequest{
				Name:      "Hyphen App",
				Subdomain: "-invalid",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate(baseDomain)
			if (err != nil) != tt.expectErr {
				t.Errorf("Validate() error = %v, expectErr = %v", err, tt.expectErr)
			}
		})
	}
}

func TestValidateCustomDomain(t *testing.T) {
	baseDomain := "yourplatform.test"

	tests := []struct {
		domain    string
		expectErr bool
	}{
		{"example.com", false},
		{"shop.mybrand.org", false},
		{"", true},
		{"yourplatform.test", true},
		{"sub.yourplatform.test", true},
		{"invalid-domain", true},
	}

	for _, tt := range tests {
		err := model.ValidateCustomDomain(tt.domain, baseDomain)
		if (err != nil) != tt.expectErr {
			t.Errorf("ValidateCustomDomain(%q) error = %v; expectErr = %v", tt.domain, err, tt.expectErr)
		}
	}
}
