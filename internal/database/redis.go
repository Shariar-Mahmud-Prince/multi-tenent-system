package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/model"
)

// RedisClient wraps go-redis with tenant caching and rate-limiting helpers.
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient initializes a connection pool to Redis.
// If Redis is unavailable, returns nil with a logged warning so the application can degrade gracefully.
func NewRedisClient(cfg *config.Config) *RedisClient {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Printf("[redis] Warning: Invalid REDIS_URL (%s): %v. Redis caching disabled.", cfg.RedisURL, err)
		return nil
	}

	opts.PoolSize = 50
	opts.MinIdleConns = 10
	opts.DialTimeout = 3 * time.Second
	opts.ReadTimeout = 2 * time.Second
	opts.WriteTimeout = 2 * time.Second

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[redis] Notice: Redis not reachable (%v). Operating in direct database mode.", err)
		return nil
	}

	log.Printf("[redis] Successfully connected to Redis instance at %s", opts.Addr)
	return &RedisClient{client: client}
}

// Close closes the underlying Redis connection pool.
func (r *RedisClient) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

// GetCachedTenant fetches a cached tenant workspace by host header.
func (r *RedisClient) GetCachedTenant(ctx context.Context, host string) (*model.Tenant, bool) {
	if r == nil || r.client == nil {
		return nil, false
	}

	key := fmt.Sprintf("tenant:host:%s", model.CleanDomain(host))
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var tenant model.Tenant
	if err := json.Unmarshal([]byte(val), &tenant); err != nil {
		return nil, false
	}

	return &tenant, true
}

// GetCachedStore is a backward-compatible alias.
func (r *RedisClient) GetCachedStore(ctx context.Context, host string) (*model.Store, bool) {
	return r.GetCachedTenant(ctx, host)
}

// SetCachedTenant caches a tenant workspace for a specified TTL.
func (r *RedisClient) SetCachedTenant(ctx context.Context, host string, tenant *model.Tenant, ttl time.Duration) {
	if r == nil || r.client == nil || tenant == nil {
		return
	}

	data, err := json.Marshal(tenant)
	if err != nil {
		return
	}

	key := fmt.Sprintf("tenant:host:%s", model.CleanDomain(host))
	_ = r.client.Set(ctx, key, data, ttl).Err()
}

// SetCachedStore is a backward-compatible alias.
func (r *RedisClient) SetCachedStore(ctx context.Context, host string, store *model.Store, ttl time.Duration) {
	r.SetCachedTenant(ctx, host, store, ttl)
}

// InvalidateTenantCache invalidates cache entries associated with a tenant's hostnames.
func (r *RedisClient) InvalidateTenantCache(ctx context.Context, baseDomain string, tenant *model.Tenant) {
	if r == nil || r.client == nil || tenant == nil {
		return
	}

	keys := []string{
		fmt.Sprintf("tenant:host:%s", tenant.FullSubdomain(baseDomain)),
	}
	if tenant.CustomDomain != nil && *tenant.CustomDomain != "" {
		keys = append(keys, fmt.Sprintf("tenant:host:%s", *tenant.CustomDomain))
	}

	_ = r.client.Del(ctx, keys...).Err()
}

// InvalidateStoreCache is a backward-compatible alias.
func (r *RedisClient) InvalidateStoreCache(ctx context.Context, baseDomain string, store *model.Store) {
	r.InvalidateTenantCache(ctx, baseDomain, store)
}

// AllowRateLimit checks if an IP is allowed under a sliding window limit using Redis.
func (r *RedisClient) AllowRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	if r == nil || r.client == nil {
		return true, nil // Fail open if Redis is not configured
	}

	redisKey := fmt.Sprintf("ratelimit:%s", key)
	count, err := r.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return true, err
	}

	if count == 1 {
		_ = r.client.Expire(ctx, redisKey, window).Err()
	}

	return count <= limit, nil
}
