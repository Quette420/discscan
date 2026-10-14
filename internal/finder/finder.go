// Package finder implements the disk analysis modes on top of the scanner:
// largest files, recently added large files and largest folders.
package finder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gsaranov/discscan/internal/scan"
)

// maxErrorSamples bounds how many read errors are kept for display.
const maxErrorSamples = 20

// Env carries settings shared by all modes.
type Env struct {
	Workers int
	// Stats, if set, receives live counters so callers can show progress.
	Stats *scan.Stats
}

// Summary describes a finished (or interrupted) scan.
type Summary struct {
	Root        string
	Counters    scan.Counters
	Elapsed     time.Duration
	Interrupted bool
	Errors      []string // up to maxErrorSamples messages
}

// FileReport is the result of a file-oriented mode.
type FileReport struct {
	Summary
	Files []scan.File
}

// ResolveRoot validates a scan root and returns its absolute form.
// A bare drive like "D:" is treated as the drive root "D:\".
func ResolveRoot(path string) (string, error) {
	if len(path) == 2 && path[1] == ':' {
		path += `\`
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("bad path %q: %w", path, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("cannot access %s: %w", abs, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", abs)
	}

	return abs, nil
}

// walk runs the scanner and fills in a Summary.
func walk(ctx context.Context, env Env, root string, opts scan.Options) (Summary, error) {
	root, err := ResolveRoot(root)
	if err != nil {
		return Summary{}, err
	}

	stats := env.Stats
	if stats == nil {
		stats = new(scan.Stats)
	}

	var (
		mu      sync.Mutex
		samples []string
	)

	opts.Workers = env.Workers
	opts.OnError = func(path string, err error) {
		mu.Lock()
		if len(samples) < maxErrorSamples {
			samples = append(samples, err.Error())
		}
		mu.Unlock()
	}

	start := time.Now()
	err = scan.Walk(ctx, root, opts, stats)

	if err != nil && !errors.Is(err, context.Canceled) {
		return Summary{}, err
	}

	return Summary{
		Root:        root,
		Counters:    stats.Snapshot(),
		Elapsed:     time.Since(start),
		Interrupted: err != nil,
		Errors:      samples,
	}, nil
}
