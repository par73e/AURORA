package responsecache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct{ Client *redis.Client }

func (s RedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := s.Client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	return data, err
}

func (s RedisStore) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return s.Client.Set(ctx, key, data, ttl).Err()
}
