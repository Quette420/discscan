package finder

import (
	"context"
	"time"

	"github.com/gsaranov/discscan/internal/scan"
	"github.com/gsaranov/discscan/internal/top"
)

// Largest finds the `limit` biggest files under root that are at least minSize bytes.
func Largest(ctx context.Context, env Env, root string, limit int, minSize int64) (*FileReport, error) {
	best := top.New(limit, func(a, b scan.File) bool {
		return a.Size < b.Size
	})

	sum, err := walk(ctx, env, root, scan.Options{
		OnFile: func(f scan.File) {
			if f.Size > 0 && f.Size >= minSize {
				best.Add(f)
			}
		},
	})
	if err != nil {
		return nil, err
	}

	return &FileReport{Summary: sum, Files: best.Sorted()}, nil
}

// Recent finds files of at least minSize bytes that appeared or changed after
// since, newest first. At most `limit` files are returned.
func Recent(ctx context.Context, env Env, root string, limit int, minSize int64, since time.Time) (*FileReport, error) {
	newest := top.New(limit, func(a, b scan.File) bool {
		return a.Changed().Before(b.Changed())
	})

	sum, err := walk(ctx, env, root, scan.Options{
		OnFile: func(f scan.File) {
			if f.Size >= minSize && !f.Changed().Before(since) {
				newest.Add(f)
			}
		},
	})
	if err != nil {
		return nil, err
	}

	return &FileReport{Summary: sum, Files: newest.Sorted()}, nil
}
