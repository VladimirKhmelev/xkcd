package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"yadro.com/course/search/core"
)

const ttl = 10 * time.Minute

type Redis struct {
	client *redis.Client
}

func New(address string) (*Redis, error) {
	client := redis.NewClient(&redis.Options{Addr: address})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Redis{client: client}, nil
}

func (r *Redis) Get(ctx context.Context, key string) ([]core.Comics, bool, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	var comics []core.Comics
	if err := json.Unmarshal(data, &comics); err != nil {
		return nil, false, err
	}

	return comics, true, nil
}

func (r *Redis) Set(ctx context.Context, key string, comics []core.Comics) error {
	data, err := json.Marshal(comics)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *Redis) Flush(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}
