package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"multi-tenant-domains/internal/model"
)

var (
	ErrStoreNotFound     = errors.New("store not found")
	ErrSubdomainTaken    = errors.New("subdomain is already taken")
	ErrCustomDomainTaken = errors.New("custom domain is already taken")
	ErrStoreInactive     = errors.New("store is inactive")
)

// StoreRepository defines database operations for tenant stores.
type StoreRepository interface {
	CreateStore(ctx context.Context, store *model.Store) error
	GetStoreByID(ctx context.Context, id uuid.UUID) (*model.Store, error)
	GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error)
	GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error)
	ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error)
	UpdateCustomDomain(ctx context.Context, id uuid.UUID, customDomain *string) error
	ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error)
	DeleteCustomDomain(ctx context.Context, id uuid.UUID) error
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
		INSERT INTO stores (name, subdomain, custom_domain, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at;
	`

	var customDomain sql.NullString
	if s.CustomDomain != nil && *s.CustomDomain != "" {
		customDomain = sql.NullString{String: *s.CustomDomain, Valid: true}
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		s.Name,
		strings.ToLower(s.Subdomain),
		customDomain,
		s.IsActive,
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
		SELECT id, name, subdomain, custom_domain, is_active, created_at, updated_at
		FROM stores
		WHERE id = $1;
	`

	var s model.Store
	var customDomain sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&s.Name,
		&s.Subdomain,
		&customDomain,
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

	if customDomain.Valid {
		s.CustomDomain = &customDomain.String
	}

	return &s, nil
}

func (r *postgresStoreRepository) GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error) {
	query := `
		SELECT id, name, subdomain, custom_domain, is_active, created_at, updated_at
		FROM stores
		WHERE LOWER(subdomain) = LOWER($1);
	`

	var s model.Store
	var customDomain sql.NullString

	err := r.db.QueryRowContext(ctx, query, strings.ToLower(subdomain)).Scan(
		&s.ID,
		&s.Name,
		&s.Subdomain,
		&customDomain,
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

	if customDomain.Valid {
		s.CustomDomain = &customDomain.String
	}

	return &s, nil
}

func (r *postgresStoreRepository) GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error) {
	query := `
		SELECT id, name, subdomain, custom_domain, is_active, created_at, updated_at
		FROM stores
		WHERE LOWER(custom_domain) = LOWER($1);
	`

	var s model.Store
	var dbCustomDomain sql.NullString

	err := r.db.QueryRowContext(ctx, query, strings.ToLower(customDomain)).Scan(
		&s.ID,
		&s.Name,
		&s.Subdomain,
		&dbCustomDomain,
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

	if dbCustomDomain.Valid {
		s.CustomDomain = &dbCustomDomain.String
	}

	return &s, nil
}

// ResolveStoreByHost resolves a tenant store from an incoming Host header.
// It first checks if the host is a platform subdomain, and otherwise falls back to custom domain.
func (r *postgresStoreRepository) ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error) {
	cleaned := model.CleanDomain(host)
	base := strings.ToLower(strings.TrimSpace(baseDomain))

	// Check if this matches <subdomain>.<baseDomain>
	if strings.HasSuffix(cleaned, "."+base) {
		subdomain := strings.TrimSuffix(cleaned, "."+base)
		// Avoid matching nested subdomains like foo.bar.yourplatform.test if not supported
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

	// Try custom domain matching
	store, err := r.GetStoreByCustomDomain(ctx, cleaned)
	if err == nil {
		return store, nil
	}

	return nil, ErrStoreNotFound
}

func (r *postgresStoreRepository) UpdateCustomDomain(ctx context.Context, id uuid.UUID, customDomain *string) error {
	query := `
		UPDATE stores
		SET custom_domain = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2;
	`

	var cd sql.NullString
	if customDomain != nil && *customDomain != "" {
		cd = sql.NullString{String: strings.ToLower(*customDomain), Valid: true}
	}

	res, err := r.db.ExecContext(ctx, query, cd, id)
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
		return ErrStoreNotFound
	}

	return nil
}

func (r *postgresStoreRepository) DeleteCustomDomain(ctx context.Context, id uuid.UUID) error {
	return r.UpdateCustomDomain(ctx, id, nil)
}

func (r *postgresStoreRepository) ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error) {
	countQuery := `SELECT COUNT(*) FROM stores;`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count stores: %w", err)
	}

	query := `
		SELECT id, name, subdomain, custom_domain, is_active, created_at, updated_at
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
		var cd sql.NullString
		if err := rows.Scan(&s.ID, &s.Name, &s.Subdomain, &cd, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan store row: %w", err)
		}
		if cd.Valid {
			s.CustomDomain = &cd.String
		}
		stores = append(stores, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating stores: %w", err)
	}

	return stores, total, nil
}
