package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	rateLimitPrefix = "rate_limit:"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(host string, port int, password string, username string, db int) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		Username: username,
		DB:       db,
	})

	// Verify Redis connection is working by sending a PING command
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
	}, nil
}

func (s *RedisStore) CheckRateLimit(ctx context.Context, ip string, route string, requestsPerHour, requestsPerMinute int) (bool, error) {
	// Check hour limit first
	hourKey := fmt.Sprintf("%s%s:%s:hour", rateLimitPrefix, ip, route)
	hourCount, err := s.client.Get(ctx, hourKey).Int64()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("failed to get hour count: %w", err)
	}
	if hourCount >= int64(requestsPerHour) {
		return false, nil
	}

	// Check minute limit
	minuteKey := fmt.Sprintf("%s%s:%s:minute", rateLimitPrefix, ip, route)
	minuteCount, err := s.client.Get(ctx, minuteKey).Int64()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("failed to get minute count: %w", err)
	}
	if minuteCount >= int64(requestsPerMinute) {
		return false, nil
	}

	// If we're under both limits, increment the counters
	pipe := s.client.Pipeline()

	// Increment hour counter
	pipe.Incr(ctx, hourKey)
	if hourCount == 0 {
		pipe.Expire(ctx, hourKey, time.Hour)
	}

	// Increment minute counter
	pipe.Incr(ctx, minuteKey)
	if minuteCount == 0 {
		pipe.Expire(ctx, minuteKey, time.Minute)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to increment rate limit counters: %w", err)
	}

	return true, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}
