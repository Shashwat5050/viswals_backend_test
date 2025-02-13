package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	redis "github.com/redis/go-redis/v9"
	"github.com/viswals_backend_test/core/models"
	"time"
)

// in redis for suitability, data is stored as a key:value where value is in JSON format.

type Redis struct {
	client *redis.Client
	ttl    time.Duration
}

func New(connectionString string, ttl time.Duration) (*Redis, error) {
	conf, err := redis.ParseURL(connectionString)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(conf)
	status := client.Ping(context.Background())

	if status.Err() != nil {
		return nil, status.Err()
	}

	return &Redis{client: client, ttl: ttl}, nil
}

func (r *Redis) Get(ctx context.Context, key string) (*models.UserDetails, error) {
	out := r.client.Get(ctx, key)

	if out.Err() != nil {
		return nil, out.Err()
	}

	res, err := out.Result()
	if err != nil {
		return nil, err
	}

	var userDetails = new(models.UserDetails)

	err = json.Unmarshal([]byte(res), userDetails)
	if err != nil {
		return nil, err
	}

	return userDetails, nil
}

func (r *Redis) Set(ctx context.Context, key string, userDetails *models.UserDetails) error {
	b, err := json.Marshal(userDetails)
	if err != nil {
		return err
	}

	out := r.client.Set(ctx, key, string(b), r.ttl)
	if out.Err() != nil {
		return out.Err()
	}

	return nil
}

func (r *Redis) SetBulk(ctx context.Context, userDetails []*models.UserDetails) error {
	var combinedErr error
	for _, userDetail := range userDetails {
		err := r.Set(ctx, fmt.Sprint(userDetail.ID), userDetail)
		if err != nil {
			errors.Join(combinedErr, err)
		}
	}
	return combinedErr
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	out := r.client.Del(ctx, key)
	if out.Err() != nil {
		return out.Err()
	}
	return nil
}
