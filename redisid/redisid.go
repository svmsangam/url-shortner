package redisid

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Generator provides ID generation backed by a Redis counter key.
// It uses Redis INCR and INCRBY commands to allocate single or block IDs.
// The zero value is not useful; use NewGenerator to construct one.
type Generator struct {
	client redis.Cmdable
	key    string
}

// NewGenerator returns a Generator that uses the provided redis.Cmdable and
// the default key "global_url_counter".
func NewGenerator(client redis.Cmdable) *Generator {
	return &Generator{client: client, key: "global_url_counter"}
}

// NewGeneratorWithKey returns a Generator that uses a custom key.
func NewGeneratorWithKey(client redis.Cmdable, key string) *Generator {
	if key == "" {
		key = "global_url_counter"
	}
	return &Generator{client: client, key: key}
}

// NextID atomically increments the counter by 1 (INCR) and returns the new value.
// Context is forwarded to the underlying Redis command.
func (g *Generator) NextID(ctx context.Context) (uint64, error) {
	if g == nil || g.client == nil {
		return 0, errors.New("redisid: nil generator or client")
	}
	v, err := g.client.Incr(ctx, g.key).Result()
	if err != nil {
		return 0, fmt.Errorf("redisid: incr failed: %w", err)
	}
	if v < 0 {
		return 0, fmt.Errorf("redisid: negative counter value: %d", v)
	}
	return uint64(v), nil
}

// NextBlock1000 atomically increments the counter by 1000 (INCRBY 1000) and
// returns the inclusive start and end IDs of the allocated block.
// Example: if INCRBY returns 2500, this function returns (1501, 2500).
func (g *Generator) NextBlock1000(ctx context.Context) (start uint64, end uint64, err error) {
	const blockSize int64 = 1000
	if g == nil || g.client == nil {
		return 0, 0, errors.New("redisid: nil generator or client")
	}
	v, err := g.client.IncrBy(ctx, g.key, blockSize).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("redisid: incrby failed: %w", err)
	}
	if v < blockSize {
		return 0, 0, fmt.Errorf("redisid: unexpected counter value: %d", v)
	}
	end = uint64(v)
	start = end - uint64(blockSize) + 1
	return start, end, nil
}
