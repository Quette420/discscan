package scan

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWalk(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "a.bin"), 100)
	writeFile(t, filepath.Join(root, "sub", "b.bin"), 200)
	writeFile(t, filepath.Join(root, "sub", "deep", "c.bin"), 300)
	writeFile(t, filepath.Join(root, "sub", "deep", "d.bin"), 400)
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	var (
		mu    sync.Mutex
		files = map[string]int64{}
		dirs  = map[string]int64{}
	)

	var stats Stats

	err := Walk(context.Background(), root, Options{
		Workers: 4,
		OnFile: func(f File) {
			mu.Lock()
			files[f.Path] = f.Size
			mu.Unlock()
		},
		OnDir: func(dir string, bytes, _ int64) {
			mu.Lock()
			dirs[dir] = bytes
			mu.Unlock()
		},
	}, &stats)
	if err != nil {
		t.Fatal(err)
	}

	c := stats.Snapshot()
	if c.Files != 4 || c.Dirs != 4 || c.Bytes != 1000 || c.Errors != 0 {
		t.Fatalf("unexpected stats: %+v", c)
	}

	if files[filepath.Join(root, "sub", "deep", "d.bin")] != 400 {
		t.Fatalf("missing or wrong file entry: %v", files)
	}

	wantDirs := map[string]int64{
		root:                               100,
		filepath.Join(root, "sub"):         200,
		filepath.Join(root, "sub", "deep"): 700,
		filepath.Join(root, "empty"):       0,
	}
	for dir, want := range wantDirs {
		if got, ok := dirs[dir]; !ok || got != want {
			t.Errorf("dir %s: got %d (seen=%v), want %d", dir, got, ok, want)
		}
	}
}

func TestWalkCancelled(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "x", "y", "z.bin"), 10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Walk(ctx, root, Options{}, nil)
	if err != context.Canceled {
		t.Fatalf("Walk() error = %v, want context.Canceled", err)
	}
}
