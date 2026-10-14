// Package scan walks a directory tree in parallel and reports every regular file.
package scan

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// File describes a regular file found during a walk.
type File struct {
	Path    string
	Size    int64
	ModTime time.Time
	Created time.Time // zero when the platform does not expose it
}

// Changed returns the later of the modification and creation times.
//
// Copied, extracted or downloaded files often keep their original
// modification time but get a fresh creation time, so this is the best
// answer to "when did this file appear on my disk".
func (f File) Changed() time.Time {
	if f.Created.After(f.ModTime) {
		return f.Created
	}
	return f.ModTime
}

// Options configures a walk. Callbacks are invoked concurrently from
// multiple goroutines and must be safe for concurrent use.
type Options struct {
	// Workers is the number of parallel directory readers (default: 2×CPU).
	Workers int

	// OnFile is called for every regular file.
	OnFile func(File)

	// OnDir is called once per directory with the total size and count of
	// the files directly inside it (not including subdirectories).
	OnDir func(dir string, bytes, files int64)

	// OnError is called for every directory or file that could not be read.
	OnError func(path string, err error)
}

// Stats holds live counters that can be read while a walk is in progress.
type Stats struct {
	Dirs   atomic.Int64
	Files  atomic.Int64
	Bytes  atomic.Int64
	Errors atomic.Int64
}

// Counters is a point-in-time copy of Stats.
type Counters struct {
	Dirs, Files, Bytes, Errors int64
}

// Snapshot returns the current counter values.
func (s *Stats) Snapshot() Counters {
	return Counters{
		Dirs:   s.Dirs.Load(),
		Files:  s.Files.Load(),
		Bytes:  s.Bytes.Load(),
		Errors: s.Errors.Load(),
	}
}

// Walk scans root recursively. Symbolic links, junctions and other reparse
// points are not followed. When ctx is cancelled the walk stops early and
// returns ctx.Err(); everything reported up to that point remains valid.
func Walk(ctx context.Context, root string, opts Options, stats *Stats) error {
	if stats == nil {
		stats = new(Stats)
	}

	workers := opts.Workers
	if workers < 1 {
		workers = runtime.NumCPU() * 2
	}

	w := &walker{
		ctx:   ctx,
		opts:  opts,
		stats: stats,
		queue: newDirQueue(root),
	}

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.run()
		}()
	}

	wg.Wait()

	return ctx.Err()
}

type walker struct {
	ctx   context.Context
	opts  Options
	stats *Stats
	queue *dirQueue
}

func (w *walker) run() {
	for {
		dir, ok := w.queue.pop()
		if !ok {
			return
		}

		// After cancellation the remaining queue is drained without reading.
		if w.ctx.Err() == nil {
			w.readDir(dir)
		}

		w.queue.done()
	}
}

func (w *walker) readDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		w.fail(dir, err)
		// ReadDir may still return the entries read before the error.
	}

	var bytes, files int64

	for _, e := range entries {
		typ := e.Type()

		// Skip symlinks, junctions and other reparse points: following them
		// would double count data and can create cycles.
		if typ&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			continue
		}

		path := filepath.Join(dir, e.Name())

		if e.IsDir() {
			w.queue.add(path)
			continue
		}

		if !typ.IsRegular() {
			continue
		}

		info, err := e.Info()
		if err != nil {
			w.fail(path, err)
			continue
		}

		size := info.Size()
		bytes += size
		files++

		if w.opts.OnFile != nil {
			w.opts.OnFile(File{
				Path:    path,
				Size:    size,
				ModTime: info.ModTime(),
				Created: createdTime(info),
			})
		}
	}

	w.stats.Dirs.Add(1)
	w.stats.Files.Add(files)
	w.stats.Bytes.Add(bytes)

	if w.opts.OnDir != nil {
		w.opts.OnDir(dir, bytes, files)
	}
}

func (w *walker) fail(path string, err error) {
	w.stats.Errors.Add(1)

	if w.opts.OnError != nil {
		w.opts.OnError(path, err)
	}
}
