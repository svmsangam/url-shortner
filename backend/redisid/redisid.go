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

// DefaultSeed is the initial counter seed used to ensure the first generated
// Base62 short code has the desired length. This mirrors the previous value
// used in cassandra.go (62^6) so that first short codes are 7 chars long.
const DefaultSeed uint64 = 56800235584

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

// EnsureSeed ensures the Redis counter key is initialized. If the key does not
// exist it will be set to seed-1 so that the first INCR returns seed. Passing
// seed==0 will use DefaultSeed.
func (g *Generator) EnsureSeed(ctx context.Context, seed uint64) error {
	if g == nil || g.client == nil {
		return errors.New("redisid: nil generator or client")
	}
	if seed == 0 {
		seed = DefaultSeed
	}
	seedMinusOne := int64(seed - 1)
	// Use SetNX to set the value only if the key does not exist.
	if _, err := g.client.SetNX(ctx, g.key, seedMinusOne, 0).Result(); err != nil {
		return fmt.Errorf("redisid: EnsureSeed SetNX failed: %w", err)
	}
	return nil
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
