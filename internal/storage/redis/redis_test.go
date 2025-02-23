package redis

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func setupTestRedis(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	port, _ := strconv.Atoi(mr.Port())
	store, err := NewRedisStore(
		mr.Host(),
		port,
		"",
		"",
		0,
	)
	if err != nil {
		mr.Close()
		t.Fatalf("Failed to create Redis store: %v", err)
	}

	return store, mr
}

func TestRateLimit(t *testing.T) {
	store, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	ip := "127.0.0.1"
	route := "test_route"

	t.Run("Check rate limit - under limit", func(t *testing.T) {
		allowed, err := store.CheckRateLimit(ctx, ip, route, 10, 5)
		if err != nil {
			t.Fatalf("Failed to check rate limit: %v", err)
		}
		if !allowed {
			t.Error("Expected request to be allowed")
		}
	})

	t.Run("Check rate limit - exceed minute limit", func(t *testing.T) {
		// Reset the state
		mr.FlushAll()

		// Make 5 requests that should be allowed
		for i := 0; i < 5; i++ {
			allowed, err := store.CheckRateLimit(ctx, ip, route, 10, 5)
			if err != nil {
				t.Fatalf("Failed to check rate limit: %v", err)
			}
			if !allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}

		// The 6th request should be blocked
		allowed, err := store.CheckRateLimit(ctx, ip, route, 10, 5)
		if err != nil {
			t.Fatalf("Failed to check rate limit: %v", err)
		}
		if allowed {
			t.Error("The 6th request should be blocked")
		}
	})

	t.Run("Check rate limit - reset after a minute", func(t *testing.T) {
		// Reset the state
		mr.FlushAll()

		// Fast forward time by 1 minute
		mr.FastForward(time.Minute)

		// Try another request
		allowed, err := store.CheckRateLimit(ctx, ip, route, 10, 5)
		if err != nil {
			t.Fatalf("Failed to check rate limit: %v", err)
		}
		if !allowed {
			t.Error("Expected request to be allowed after rate limit reset")
		}
	})

	t.Run("Check rate limit - different routes", func(t *testing.T) {
		// Reset the state
		mr.FlushAll()

		route1 := "route1"
		route2 := "route2"

		// Make 5 requests on route1
		for i := 0; i < 5; i++ {
			allowed, err := store.CheckRateLimit(ctx, ip, route1, 10, 5)
			if err != nil {
				t.Fatalf("Failed to check rate limit: %v", err)
			}
			if !allowed {
				t.Errorf("Request %d on route1 should be allowed", i+1)
			}
		}

		// Try a request on route2 (should be allowed)
		allowed, err := store.CheckRateLimit(ctx, ip, route2, 10, 5)
		if err != nil {
			t.Fatalf("Failed to check rate limit: %v", err)
		}
		if !allowed {
			t.Error("Request on route2 should be allowed")
		}
	})
}
