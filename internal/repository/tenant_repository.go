package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"multi-tenant-domains/internal/model"
)

var (
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrStoreNotFound      = ErrTenantNotFound // Backward compatibility
	ErrSubdomainTaken     = errors.New("subdomain is already taken")
	ErrCustomDomainTaken  = errors.New("custom domain is already taken")
	ErrTenantInactive     = errors.New("tenant is inactive")
	ErrStoreInactive      = ErrTenantInactive // Backward compatibility
	ErrUnauthorizedTenant = errors.New("user does not have ownership access to this tenant")
	ErrUnauthorizedStore  = ErrUnauthorizedTenant // Backward compatibility
)

// TenantRepository defines database operations for multi-tenant SaaS application workspaces.
type TenantRepository interface {
	CreateTenant(ctx context.Context, tenant *model.Tenant) error
	GetTenantByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	GetTenantBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error)
	GetTenantByCustomDomain(ctx context.Context, customDomain string) (*model.Tenant, error)
	ResolveTenantByHost(ctx context.Context, host string, baseDomain string) (*model.Tenant, error)
	UpdateCustomDomainWithVerification(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID, customDomain string, token string, expiresAt time.Time) error
	UpdateDomainStatus(ctx context.Context, id uuid.UUID, status string) error
	ListTenants(ctx context.Context, limit, offset int) ([]*model.Tenant, int, error)
	ListTenantsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Tenant, error)
	DeleteCustomDomain(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID) error

	// Backward compatibility alias methods
	CreateStore(ctx context.Context, store *model.Store) error
	GetStoreByID(ctx context.Context, id uuid.UUID) (*model.Store, error)
	GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error)
	GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error)
	ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error)
	ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error)
	ListStoresByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Store, error)
}

// Backward compatibility alias
type StoreRepository = TenantRepository

type postgresTenantRepository struct {
	db *sql.DB
}

// NewPostgresTenantRepository instantiates a new TenantRepository backed by PostgreSQL.
func NewPostgresTenantRepository(db *sql.DB) TenantRepository {
	return &postgresTenantRepository{db: db}
}

// NewPostgresStoreRepository is an alias for NewPostgresTenantRepository.
func NewPostgresStoreRepository(db *sql.DB) StoreRepository {
	return NewPostgresTenantRepository(db)
}

func (r *postgresTenantRepository) CreateTenant(ctx context.Context, t *model.Tenant) error {
	query := `
		INSERT INTO stores (name, subdomain, custom_domain, domain_status, is_active, owner_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at;
	`

	var customDomain sql.NullString
	if t.CustomDomain != nil && *t.CustomDomain != "" {
		customDomain = sql.NullString{String: *t.CustomDomain, Valid: true}
	}

	if t.DomainStatus == "" {
		t.DomainStatus = model.DomainStatusActive
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		t.Name,
		strings.ToLower(t.Subdomain),
		customDomain,
		t.DomainStatus,
		t.IsActive,
		t.OwnerID,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Constraint {
			case "idx_stores_subdomain_unique_lower", "stores_subdomain_key":
				return ErrSubdomainTaken
			case "idx_stores_custom_domain_unique_lower", "stores_custom_domain_key":
				return ErrCustomDomainTaken
			}
		}
		return fmt.Errorf("failed to insert tenant: %w", err)
	}

	return nil
}

func (r *postgresTenantRepository) CreateStore(ctx context.Context, store *model.Store) error {
	return r.CreateTenant(ctx, store)
}

func (r *postgresTenantRepository) GetTenantByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	query := `
		SELECT id, owner_id, name, subdomain, custom_domain, domain_status, 
		       verification_token, verification_expires_at, is_active, created_at, updated_at
		FROM stores
		WHERE id = $1;
	`

	var t model.Tenant
	var ownerID sql.NullString
	var customDomain sql.NullString
	var verifyToken sql.NullString
	var verifyExpires sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&ownerID,
		&t.Name,
		&t.Subdomain,
		&customDomain,
		&t.DomainStatus,
		&verifyToken,
		&verifyExpires,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to get tenant by id: %w", err)
	}

	if ownerID.Valid {
		parsed, _ := uuid.Parse(ownerID.String)
		t.OwnerID = &parsed
	}
	if customDomain.Valid {
		t.CustomDomain = &customDomain.String
	}
	if verifyToken.Valid {
		t.VerificationToken = &verifyToken.String
	}
	if verifyExpires.Valid {
		t.VerificationExpiresAt = &verifyExpires.Time
	}

	return &t, nil
}

func (r *postgresTenantRepository) GetStoreByID(ctx context.Context, id uuid.UUID) (*model.Store, error) {
	return r.GetTenantByID(ctx, id)
}

func (r *postgresTenantRepository) GetTenantBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error) {
	query := `
		SELECT id, owner_id, name, subdomain, custom_domain, domain_status, 
		       verification_token, verification_expires_at, is_active, created_at, updated_at
		FROM stores
		WHERE LOWER(subdomain) = LOWER($1);
	`

	var t model.Tenant
	var ownerID sql.NullString
	var customDomain sql.NullString
	var verifyToken sql.NullString
	var verifyExpires sql.NullTime

	err := r.db.QueryRowContext(ctx, query, strings.ToLower(subdomain)).Scan(
		&t.ID,
		&ownerID,
		&t.Name,
		&t.Subdomain,
		&customDomain,
		&t.DomainStatus,
		&verifyToken,
		&verifyExpires,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to get tenant by subdomain: %w", err)
	}

	if ownerID.Valid {
		parsed, _ := uuid.Parse(ownerID.String)
		t.OwnerID = &parsed
	}
	if customDomain.Valid {
		t.CustomDomain = &customDomain.String
	}
	if verifyToken.Valid {
		t.VerificationToken = &verifyToken.String
	}
	if verifyExpires.Valid {
		t.VerificationExpiresAt = &verifyExpires.Time
	}

	return &t, nil
}

func (r *postgresTenantRepository) GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error) {
	return r.GetTenantBySubdomain(ctx, subdomain)
}

func (r *postgresTenantRepository) GetTenantByCustomDomain(ctx context.Context, customDomain string) (*model.Tenant, error) {
	query := `
		SELECT id, owner_id, name, subdomain, custom_domain, domain_status, 
		       verification_token, verification_expires_at, is_active, created_at, updated_at
		FROM stores
		WHERE LOWER(custom_domain) = LOWER($1);
	`

	var t model.Tenant
	var ownerID sql.NullString
	var dbCustomDomain sql.NullString
	var verifyToken sql.NullString
	var verifyExpires sql.NullTime

	err := r.db.QueryRowContext(ctx, query, strings.ToLower(customDomain)).Scan(
		&t.ID,
		&ownerID,
		&t.Name,
		&t.Subdomain,
		&dbCustomDomain,
		&t.DomainStatus,
		&verifyToken,
		&verifyExpires,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to get tenant by custom domain: %w", err)
	}

	if ownerID.Valid {
		parsed, _ := uuid.Parse(ownerID.String)
		t.OwnerID = &parsed
	}
	if dbCustomDomain.Valid {
		t.CustomDomain = &dbCustomDomain.String
	}
	if verifyToken.Valid {
		t.VerificationToken = &verifyToken.String
	}
	if verifyExpires.Valid {
		t.VerificationExpiresAt = &verifyExpires.Time
	}

	return &t, nil
}

func (r *postgresTenantRepository) GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error) {
	return r.GetTenantByCustomDomain(ctx, customDomain)
}

// ResolveTenantByHost resolves a tenant application workspace from an incoming Host header.
func (r *postgresTenantRepository) ResolveTenantByHost(ctx context.Context, host string, baseDomain string) (*model.Tenant, error) {
	cleaned := model.CleanDomain(host)
	base := strings.ToLower(strings.TrimSpace(baseDomain))

	// Check if this matches <subdomain>.<baseDomain>
	if strings.HasSuffix(cleaned, "."+base) {
		subdomain := strings.TrimSuffix(cleaned, "."+base)
		if !strings.Contains(subdomain, ".") {
			tenant, err := r.GetTenantBySubdomain(ctx, subdomain)
			if err == nil {
				return tenant, nil
			}
			if !errors.Is(err, ErrTenantNotFound) {
				return nil, err
			}
		}
	}

	// Try custom domain matching (only allows verified, active, or provisioning status)
	tenant, err := r.GetTenantByCustomDomain(ctx, cleaned)
	if err == nil {
		if tenant.DomainStatus == model.DomainStatusPending || tenant.DomainStatus == model.DomainStatusFailed {
			return nil, ErrTenantNotFound
		}
		return tenant, nil
	}

	return nil, ErrTenantNotFound
}

func (r *postgresTenantRepository) ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error) {
	return r.ResolveTenantByHost(ctx, host, baseDomain)
}

// UpdateCustomDomainWithVerification sets domain into pending_verification and stores DNS token.
func (r *postgresTenantRepository) UpdateCustomDomainWithVerification(
	ctx context.Context,
	id uuid.UUID,
	ownerID *uuid.UUID,
	customDomain string,
	token string,
	expiresAt time.Time,
) error {
	query := `
		UPDATE stores
		SET custom_domain = $1, 
		    domain_status = $2, 
		    verification_token = $3, 
		    verification_expires_at = $4, 
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND ($6::uuid IS NULL OR owner_id = $6);
	`

	res, err := r.db.ExecContext(
		ctx,
		query,
		strings.ToLower(customDomain),
		model.DomainStatusPending,
		token,
		expiresAt,
		id,
		ownerID,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Constraint == "idx_stores_custom_domain_unique_lower" || pqErr.Constraint == "stores_custom_domain_key" {
				return ErrCustomDomainTaken
			}
		}
		return fmt.Errorf("failed to update custom domain: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUnauthorizedTenant
	}

	return nil
}

func (r *postgresTenantRepository) UpdateDomainStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE stores
		SET domain_status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2;
	`
	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTenantNotFound
	}
	return nil
}

func (r *postgresTenantRepository) DeleteCustomDomain(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID) error {
	query := `
		UPDATE stores
		SET custom_domain = NULL, 
		    domain_status = $1, 
		    verification_token = NULL, 
		    verification_expires_at = NULL, 
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND ($3::uuid IS NULL OR owner_id = $3);
	`
	res, err := r.db.ExecContext(ctx, query, model.DomainStatusActive, id, ownerID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUnauthorizedTenant
	}
	return nil
}

func (r *postgresTenantRepository) ListTenants(ctx context.Context, limit, offset int) ([]*model.Tenant, int, error) {
	countQuery := `SELECT COUNT(*) FROM stores;`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count tenants: %w", err)
	}

	query := `
		SELECT id, owner_id, name, subdomain, custom_domain, domain_status, 
		       verification_token, verification_expires_at, is_active, created_at, updated_at
		FROM stores
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*model.Tenant
	for rows.Next() {
		var t model.Tenant
		var ownerID sql.NullString
		var cd sql.NullString
		var vt sql.NullString
		var ve sql.NullTime

		if err := rows.Scan(
			&t.ID,
			&ownerID,
			&t.Name,
			&t.Subdomain,
			&cd,
			&t.DomainStatus,
			&vt,
			&ve,
			&t.IsActive,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan tenant row: %w", err)
		}

		if ownerID.Valid {
			p, _ := uuid.Parse(ownerID.String)
			t.OwnerID = &p
		}
		if cd.Valid {
			t.CustomDomain = &cd.String
		}
		if vt.Valid {
			t.VerificationToken = &vt.String
		}
		if ve.Valid {
			t.VerificationExpiresAt = &ve.Time
		}

		tenants = append(tenants, &t)
	}

	return tenants, total, nil
}

func (r *postgresTenantRepository) ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error) {
	return r.ListTenants(ctx, limit, offset)
}

func (r *postgresTenantRepository) ListTenantsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Tenant, error) {
	query := `
		SELECT id, owner_id, name, subdomain, custom_domain, domain_status, 
		       verification_token, verification_expires_at, is_active, created_at, updated_at
		FROM stores
		WHERE owner_id = $1
		ORDER BY created_at DESC;
	`

	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants by owner: %w", err)
	}
	defer rows.Close()

	var tenants []*model.Tenant
	for rows.Next() {
		var t model.Tenant
		var oID sql.NullString
		var cd sql.NullString
		var vt sql.NullString
		var ve sql.NullTime

		if err := rows.Scan(
			&t.ID,
			&oID,
			&t.Name,
			&t.Subdomain,
			&cd,
			&t.DomainStatus,
			&vt,
			&ve,
			&t.IsActive,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		if oID.Valid {
			p, _ := uuid.Parse(oID.String)
			t.OwnerID = &p
		}
		if cd.Valid {
			t.CustomDomain = &cd.String
		}
		if vt.Valid {
			t.VerificationToken = &vt.String
		}
		if ve.Valid {
			t.VerificationExpiresAt = &ve.Time
		}

		tenants = append(tenants, &t)
	}

	return tenants, nil
}

func (r *postgresTenantRepository) ListStoresByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Store, error) {
	return r.ListTenantsByOwner(ctx, ownerID)
}
