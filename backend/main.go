package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gocql/gocql"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"url-shortner/redisid"
	"url-shortner/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: could not load root .env file; using environment variables and defaults: %v", err)
	}

	// Configuration from environment with sensible defaults
	port := getEnv("PORT", "8080")
	cassandraHosts := getEnv("CASSANDRA_HOSTS", "127.0.0.1")
	keyspace := getEnv("CASSANDRA_KEYSPACE", "urlshortener")
	redisAddr := getEnv("REDIS_ADDR", "127.0.0.1:6379")

	// Initialize Cassandra session
	cluster := gocql.NewCluster(strings.Split(cassandraHosts, ",")...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.LocalOne // Use Quorum instead of LocalOne if multipel-NDC for speed
	cluster.NumConns = 10                // Increase streams per host (Default: 2)
	cluster.ConnectTimeout = 5 * time.Second
	cluster.Timeout = 3 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("failed to create Cassandra session: %v", err)
	}
	// Ensure session is closed on exit
	defer session.Close()

	db := store.NewDB(session)
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("warning: error closing db: %v", err)
		}
	}()

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		PoolSize:     300,             // Scale connection pool for high concurrency
		MinIdleConns: 50,              // Keep warm connections ready to eliminate handshake delay
		DialTimeout:  2 * time.Second, // Fail fast if Redis is unresponsive
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		// You can add Password/DB here if needed via environment variables
	})
	// Test Redis connectivity (optional but helpful at startup)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to ping redis at %s: %v", redisAddr, err)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Printf("warning: error closing redis client: %v", err)
		}
	}()

	// Create a Redis-backed ID generator and ensure the counter is seeded
	gen := redisid.NewGenerator(rdb)
	if err := gen.EnsureSeed(context.Background(), 0); err != nil {
		log.Fatalf("failed to seed redis id generator: %v", err)
	}

	// Build router (routes and middleware are separated)
	router := SetupRouter(db, gen)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in background
	go func() {
		log.Printf("starting server on %s (Cassandra=%s keyspace=%s, Redis=%s)", srv.Addr, cassandraHosts, keyspace, redisAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("shutting down server...")

	// Context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	} else {
		log.Println("server stopped gracefully")
	}
}

// getEnv reads an environment variable and returns fallback if unset.
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
