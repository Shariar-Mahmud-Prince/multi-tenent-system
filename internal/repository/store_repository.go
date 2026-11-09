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
	ErrStoreNotFound     = errors.New("store not found")
	ErrSubdomainTaken    = errors.New("subdomain is already taken")
	ErrCustomDomainTaken = errors.New("custom domain is already taken")
	ErrStoreInactive     = errors.New("store is inactive")
	ErrUnauthorizedStore = errors.New("user does not have ownership access to this store")
)

// StoreRepository defines database operations for tenant stores.
type StoreRepository interface {
	CreateStore(ctx context.Context, store *model.Store) error
	GetStoreByID(ctx context.Context, id uuid.UUID) (*model.Store, error)
	GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error)
	GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error)
	ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error)
	UpdateCustomDomainWithVerification(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID, customDomain string, token string, expiresAt time.Time) error
	UpdateDomainStatus(ctx context.Context, id uuid.UUID, status string) error
	ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error)
	ListStoresByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Store, error)
	DeleteCustomDomain(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID) error
}

type postgresStoreRepository struct {
	db *sql.DB
}

// NewPostgresStoreRepository instantiates a new StoreRepository backed by PostgreSQL.
func NewPostgresStoreRepository(db *sql.DB) StoreRepository {
	return &postgresStoreRepository{db: db}
}

func (r *postgresStoreRepository) CreateStore(ctx context.Context, s *model.Store) error {
	query := `
		INSERT INTO stores (name, subdomain, custom_domain, domain_status, is_active, owner_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at;
	`

	var customDomain sql.NullString
	if s.CustomDomain != nil && *s.CustomDomain != "" {
		customDomain = sql.NullString{String: *s.CustomDomain, Valid: true}
	}

	if s.DomainStatus == "" {
		s.DomainStatus = model.DomainStatusActive
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		s.Name,
		strings.ToLower(s.Subdomain),
		customDomain,
		s.DomainStatus,
		s.IsActive,
		s.OwnerID,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Constraint {
			case "idx_stores_subdomain_unique_lower", "stores_subdomain_key":
				return ErrSubdomainTaken
			case "idx_stores_custom_domain_unique_lower", "stores_custom_domain_key":
				return ErrCustomDomainTaken
			}
		}
		return fmt.Errorf("failed to insert store: %w", err)
	}

	return nil
}

func (r *postgresStoreRepository) GetStoreByID(ctx context.Context, id uuid.UUID) (*model.Store, error) {
	query := `
		SELECT id, owner_id, name, subdomain, custom_domain, domain_status, 
		       verification_token, verification_expires_at, is_active, created_at, updated_at
		FROM stores
		WHERE id = $1;
	`

	var s model.Store
	var ownerID sql.NullString
	var customDomain sql.NullString
	var verifyToken sql.NullString
	var verifyExpires sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&ownerID,
		&s.Name,
		&s.Subdomain,
		&customDomain,
		&s.DomainStatus,
		&verifyToken,
		&verifyExpires,
		&s.IsActive,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStoreNotFound
		}
		return nil, fmt.Errorf("failed to get store by id: %w", err)
	}

	if ownerID.Valid {
		parsed, _ := uuid.Parse(ownerID.String)
		s.OwnerID = &parsed
	}
	if customDomain.Valid {
		s.CustomDomain = &customDomain.String
	}
	if verifyToken.Valid {
		s.VerificationToken = &verifyToken.String
	}
	if verifyExpires.Valid {
		s.VerificationExpiresAt = &verifyExpires.Time
	}

	return &s, nil
}

func (r *postgresStoreRepository) GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error) {
	query := `
		SELECT id, owner_id, name, subdomain, custom_domain, domain_status, 
		       verification_token, verification_expires_at, is_active, created_at, updated_at
		FROM stores
		WHERE LOWER(subdomain) = LOWER($1);
	`

	var s model.Store
	var ownerID sql.NullString
	var customDomain sql.NullString
	var verifyToken sql.NullString
	var verifyExpires sql.NullTime

	err := r.db.QueryRowContext(ctx, query, strings.ToLower(subdomain)).Scan(
		&s.ID,
		&ownerID,
		&s.Name,
		&s.Subdomain,
		&customDomain,
		&s.DomainStatus,
		&verifyToken,
		&verifyExpires,
		&s.IsActive,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStoreNotFound
		}
		return nil, fmt.Errorf("failed to get store by subdomain: %w", err)
	}

	if ownerID.Valid {
		parsed, _ := uuid.Parse(ownerID.String)
		s.OwnerID = &parsed
	}
	if customDomain.Valid {
		s.CustomDomain = &customDomain.String
	}
	if verifyToken.Valid {
		s.VerificationToken = &verifyToken.String
	}
	if verifyExpires.Valid {
		s.VerificationExpiresAt = &verifyExpires.Time
	}

	return &s, nil
}

func (r *postgresStoreRepository) GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error) {
	query := `
		SELECT id, owner_id, name, subdomain, custom_domain, domain_status, 
		       verification_token, verification_expires_at, is_active, created_at, updated_at
		FROM stores
		WHERE LOWER(custom_domain) = LOWER($1);
	`

	var s model.Store
	var ownerID sql.NullString
	var dbCustomDomain sql.NullString
	var verifyToken sql.NullString
	var verifyExpires sql.NullTime

	err := r.db.QueryRowContext(ctx, query, strings.ToLower(customDomain)).Scan(
		&s.ID,
		&ownerID,
		&s.Name,
		&s.Subdomain,
		&dbCustomDomain,
		&s.DomainStatus,
		&verifyToken,
		&verifyExpires,
		&s.IsActive,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStoreNotFound
		}
		return nil, fmt.Errorf("failed to get store by custom domain: %w", err)
	}

	if ownerID.Valid {
		parsed, _ := uuid.Parse(ownerID.String)
		s.OwnerID = &parsed
	}
	if dbCustomDomain.Valid {
		s.CustomDomain = &dbCustomDomain.String
	}
	if verifyToken.Valid {
		s.VerificationToken = &verifyToken.String
	}
	if verifyExpires.Valid {
		s.VerificationExpiresAt = &verifyExpires.Time
	}

	return &s, nil
}

// ResolveStoreByHost resolves a tenant store from an incoming Host header.
func (r *postgresStoreRepository) ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error) {
	cleaned := model.CleanDomain(host)
	base := strings.ToLower(strings.TrimSpace(baseDomain))

	// Check if this matches <subdomain>.<baseDomain>
	if strings.HasSuffix(cleaned, "."+base) {
		subdomain := strings.TrimSuffix(cleaned, "."+base)
		if !strings.Contains(subdomain, ".") {
			store, err := r.GetStoreBySubdomain(ctx, subdomain)
			if err == nil {
				return store, nil
			}
			if !errors.Is(err, ErrStoreNotFound) {
				return nil, err
			}
		}
	}

	// Try custom domain matching (only allows verified, active, or provisioning status)
	store, err := r.GetStoreByCustomDomain(ctx, cleaned)
	if err == nil {
		// Custom domains must not be in pending_verification or failed status for public traffic
		if store.DomainStatus == model.DomainStatusPending || store.DomainStatus == model.DomainStatusFailed {
			return nil, ErrStoreNotFound
		}
		return store, nil
	}

	return nil, ErrStoreNotFound
}

// UpdateCustomDomainWithVerification sets domain into pending_verification and stores DNS token.
// Enforces tenant ownership checks to prevent IDOR attacks.
func (r *postgresStoreRepository) UpdateCustomDomainWithVerification(
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
		return ErrUnauthorizedStore
	}

	return nil
}

func (r *postgresStoreRepository) UpdateDomainStatus(ctx context.Context, id uuid.UUID, status string) error {
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
		return ErrStoreNotFound
	}
	return nil
}

func (r *postgresStoreRepository) DeleteCustomDomain(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID) error {
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
		return ErrUnauthorizedStore
	}
	return nil
}

func (r *postgresStoreRepository) ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error) {
	countQuery := `SELECT COUNT(*) FROM stores;`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count stores: %w", err)
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
		return nil, 0, fmt.Errorf("failed to list stores: %w", err)
	}
	defer rows.Close()

	var stores []*model.Store
	for rows.Next() {
		var s model.Store
		var ownerID sql.NullString
		var cd sql.NullString
		var vt sql.NullString
		var ve sql.NullTime

		if err := rows.Scan(
			&s.ID,
			&ownerID,
			&s.Name,
			&s.Subdomain,
			&cd,
			&s.DomainStatus,
			&vt,
			&ve,
			&s.IsActive,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan store row: %w", err)
		}

		if ownerID.Valid {
			p, _ := uuid.Parse(ownerID.String)
			s.OwnerID = &p
		}
		if cd.Valid {
			s.CustomDomain = &cd.String
		}
		if vt.Valid {
			s.VerificationToken = &vt.String
		}
		if ve.Valid {
			s.VerificationExpiresAt = &ve.Time
		}

		stores = append(stores, &s)
	}

	return stores, total, nil
}

func (r *postgresStoreRepository) ListStoresByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Store, error) {
	query := `
		SELECT id, owner_id, name, subdomain, custom_domain, domain_status, 
		       verification_token, verification_expires_at, is_active, created_at, updated_at
		FROM stores
		WHERE owner_id = $1
		ORDER BY created_at DESC;
	`

	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stores by owner: %w", err)
	}
	defer rows.Close()

	var stores []*model.Store
	for rows.Next() {
		var s model.Store
		var oID sql.NullString
		var cd sql.NullString
		var vt sql.NullString
		var ve sql.NullTime

		if err := rows.Scan(
			&s.ID,
			&oID,
			&s.Name,
			&s.Subdomain,
			&cd,
			&s.DomainStatus,
			&vt,
			&ve,
			&s.IsActive,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan store: %w", err)
		}

		if oID.Valid {
			p, _ := uuid.Parse(oID.String)
			s.OwnerID = &p
		}
		if cd.Valid {
			s.CustomDomain = &cd.String
		}
		if vt.Valid {
			s.VerificationToken = &vt.String
		}
		if ve.Valid {
			s.VerificationExpiresAt = &ve.Time
		}

		stores = append(stores, &s)
	}

	return stores, nil
}
