package memory_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"multi-tenant-domains/internal/middleware"
	"multi-tenant-domains/internal/model"
)

type dummyStoreRepo struct{}

func (d *dummyStoreRepo) CreateStore(ctx context.Context, store *model.Store) error { return nil }
func (d *dummyStoreRepo) GetStoreByID(ctx context.Context, id uuid.UUID) (*model.Store, error) {
	cd := "leak.store.com"
	return &model.Store{
		ID:           id,
		Name:         "Leak Test Store",
		Subdomain:    "leak",
		CustomDomain: &cd,
		IsActive:     true,
	}, nil
}
func (d *dummyStoreRepo) GetStoreBySubdomain(ctx context.Context, subdomain string) (*model.Store, error) {
	return d.GetStoreByID(ctx, uuid.New())
}
func (d *dummyStoreRepo) GetStoreByCustomDomain(ctx context.Context, customDomain string) (*model.Store, error) {
	return d.GetStoreByID(ctx, uuid.New())
}
func (d *dummyStoreRepo) ResolveStoreByHost(ctx context.Context, host string, baseDomain string) (*model.Store, error) {
	return d.GetStoreByID(ctx, uuid.New())
}
func (d *dummyStoreRepo) UpdateCustomDomainWithVerification(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID, customDomain string, token string, expiresAt time.Time) error {
	return nil
}
func (d *dummyStoreRepo) UpdateDomainStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}
func (d *dummyStoreRepo) DeleteCustomDomain(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID) error {
	return nil
}
func (d *dummyStoreRepo) ListStores(ctx context.Context, limit, offset int) ([]*model.Store, int, error) {
	return nil, 0, nil
}
func (d *dummyStoreRepo) ListStoresByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Store, error) {
	return nil, nil
}

func TestMemory_NoLeakUnderSustainedRequests(t *testing.T) {
	repo := &dummyStoreRepo{}
	mw := middleware.TenantMiddleware(repo, "saas.local", []string{"localhost"})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Warmup and baseline
	runtime.GC()
	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)

	// Process 50,000 rapid requests
	iterations := 50000
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "leak.saas.local"

	for i := 0; i < iterations; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// Trigger GC to clear transient allocations
	runtime.GC()
	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)

	// Memory difference should be negligible (less than 5MB retained heap for 50,000 requests)
	heapDiff := int64(mAfter.HeapAlloc) - int64(mBefore.HeapAlloc)
	const maxAllowedLeakBytes = 5 * 1024 * 1024 // 5 MB

	t.Logf("Heap before: %d KB, Heap after: %d KB, Diff: %d KB across %d requests",
		mBefore.HeapAlloc/1024, mAfter.HeapAlloc/1024, heapDiff/1024, iterations)

	if heapDiff > maxAllowedLeakBytes {
		t.Fatalf("POTENTIAL MEMORY LEAK DETECTED: Heap retained %d bytes after %d requests", heapDiff, iterations)
	}
}
