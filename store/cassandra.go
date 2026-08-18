package store

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"

	"url-shortner/base62"
	"url-shortner/redisid"
)

// URLMapping maps to the Cassandra `urls` table.
type URLMapping struct {
	ShortCode   string    `db:"short_code"`
	LongURL     string    `db:"long_url"`
	DeviceToken string    `db:"device_token"`
	CreatedAt   time.Time `db:"created_at"`
}

// DeviceURL represents a row in the device_urls table.
type DeviceURL struct {
	DeviceToken string    `json:"device_token"`
	ShortCode   string    `json:"short_code"`
	LongURL     string    `json:"long_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// DB wraps a gocql session for Cassandra operations.
type DB struct {
	session *gocql.Session
}

// NewDB returns a DB instance wrapping the provided *gocql.Session (from
// github.com/gocql/gocql).
func NewDB(gocqlSession *gocql.Session) *DB {
	return &DB{session: gocqlSession}
}

// Close the underlying session when done.
func (db *DB) Close() error {
	if db == nil || db.session == nil {
		return nil
	}
	db.session.Close()
	return nil
}

// SaveURL writes the mapping into both urls and device_urls using a logged batch.
func (s *DB) SaveURL(ctx context.Context, shortCode, longURL, deviceToken string) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("db: nil session")
	}

	now := time.Now().UTC()
	batch := s.session.NewBatch(gocql.LoggedBatch)
	// Attach context if supported by gocql Batch
	if bc, ok := interface{}(batch).(interface{ WithContext(context.Context) *gocql.Batch }); ok {
		batch = bc.WithContext(ctx)
	}

	// Insert into urls (lookup by short code)
	batch.Query(
		`INSERT INTO urlshortener.urls (short_code, long_url, device_token, created_at) VALUES (?, ?, ?, ?)`,
		shortCode, longURL, deviceToken, now,
	)

	// Insert into device_urls (lookup/count by device token)
	batch.Query(
		`INSERT INTO urlshortener.device_urls (device_token, short_code, long_url, created_at) VALUES (?, ?, ?, ?)`,
		deviceToken, shortCode, longURL, now,
	)

	return s.session.ExecuteBatch(batch)
}

// CreateShortLink generates a numeric ID via Redis, encodes it as Base62 to
// obtain a 7+ char short code, persists the mapping using SaveURL and returns
// the URLMapping (no UUID primary key used any more).
func (db *DB) CreateShortLink(ctx context.Context, gen *redisid.Generator, longURL, deviceToken string) (*URLMapping, error) {
	if db == nil || db.session == nil {
		return nil, fmt.Errorf("db: nil session")
	}
	if gen == nil {
		return nil, fmt.Errorf("db: nil redis id generator")
	}

	// Ensure the redis counter is seeded (if needed). Generator exposes EnsureSeed.
	if err := gen.EnsureSeed(ctx, 0); err != nil {
		return nil, fmt.Errorf("failed to ensure redis id seed: %w", err)
	}

	// Atomically increment to get the next numeric ID
	v, err := gen.NextID(ctx)
	if err != nil {
		return nil, fmt.Errorf("redis next id failed: %w", err)
	}
	idNum := v

	// Encode to Base62
	short := base64OrBase62Encode(idNum)
	// Just in case, ensure length at least 7; if not, continue incrementing until it is.
	for len(short) < 7 {
		v, err = gen.NextID(ctx)
		if err != nil {
			return nil, fmt.Errorf("redis next id failed while ensuring length: %w", err)
		}
		idNum = v
		short = base64OrBase62Encode(idNum)
	}

	mapping := URLMapping{
		ShortCode:   short,
		LongURL:     longURL,
		DeviceToken: deviceToken,
		CreatedAt:   time.Now().UTC(),
	}

	// Persist into both tables using a logged batch
	if err := db.SaveURL(ctx, mapping.ShortCode, mapping.LongURL, mapping.DeviceToken); err != nil {
		return nil, fmt.Errorf("failed to save url mapping: %w", err)
	}

	return &mapping, nil
}

// GetLongURLByShortCode queries Cassandra for the long URL matching the given
// short code. If no row is found, gocql.ErrNotFound is returned.
func (db *DB) GetLongURLByShortCode(ctx context.Context, shortCode string) (string, error) {
	if db == nil || db.session == nil {
		return "", fmt.Errorf("db: nil session")
	}
	var longURL string
	q := db.session.Query("SELECT long_url FROM urlshortener.urls WHERE short_code = ? LIMIT 1", shortCode).WithContext(ctx)
	if err := q.Scan(&longURL); err != nil {
		return "", err
	}
	return longURL, nil
}

// helper wrapper for base62 encoding (kept indirection in case of future change)
func base64OrBase62Encode(n uint64) string {
	// existing base62.Encode expects uint64
	return base62.Encode(n)
}
