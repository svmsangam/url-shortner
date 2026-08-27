package redisid

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const cachePrefix = "shorturl:"

// cacheKey returns the Redis key for a given short code.
func (g *Generator) cacheKey(code string) string {
	return cachePrefix + code
}

// GetURL fetches the long URL associated with the provided short code from Redis.
// Returns ("", nil) when the key is not present.
func (g *Generator) GetURL(ctx context.Context, code string) (string, error) {
	if g == nil || g.client == nil {
			return "", errors.New("redisid: nil generator or client")
	}
	val, err := g.client.Get(ctx, g.cacheKey(code)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetURL caches the mapping code -> longURL with the provided TTL. If ttl==0
// a sensible default of 24 hours is used.
func (g *Generator) SetURL(ctx context.Context, code string, longURL string, ttl time.Duration) error {
	if g == nil || g.client == nil {
			return errors.New("redisid: nil generator or client")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return g.client.Set(ctx, g.cacheKey(code), longURL, ttl).Err()
}
