package storage

import "context"

type Backend interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
	Name() string
}
