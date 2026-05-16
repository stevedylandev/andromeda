package storage

import (
	"context"
	"os"
	"path/filepath"
)

type Local struct {
	Dir string
}

func NewLocal(dir string) *Local { return &Local{Dir: dir} }

func (l *Local) Put(ctx context.Context, key, contentType string, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := os.MkdirAll(l.Dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(l.Dir, key), data, 0o644)
}

func (l *Local) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	err := os.Remove(filepath.Join(l.Dir, key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (l *Local) PublicURL(key string) string { return "/uploads/" + key }
func (l *Local) Name() string                { return "local" }
