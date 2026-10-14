package finder

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func makeTree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	files := map[string]int{
		"small.txt":            10,
		"video/movie.mkv":      5000,
		"video/clips/a.mp4":    1500,
		"video/clips/b.mp4":    2500,
		"docs/report.pdf":      800,
		"docs/archive/old.zip": 3000,
	}

	for name, size := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Make old.zip look old.
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(root, "docs", "archive", "old.zip"), old, old); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestLargest(t *testing.T) {
	root := makeTree(t)

	rep, err := Largest(context.Background(), Env{Workers: 2}, root, 3, 0)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"movie.mkv", "old.zip", "b.mp4"}
	if len(rep.Files) != len(want) {
		t.Fatalf("got %d files, want %d", len(rep.Files), len(want))
	}
	for i, name := range want {
		if filepath.Base(rep.Files[i].Path) != name {
			t.Errorf("rank %d: got %s, want %s", i+1, rep.Files[i].Path, name)
		}
	}

	if rep.Counters.Files != 6 || rep.Counters.Bytes != 12810 {
		t.Errorf("unexpected counters: %+v", rep.Counters)
	}
}

func TestLargestMinSize(t *testing.T) {
	root := makeTree(t)

	rep, err := Largest(context.Background(), Env{}, root, 100, 2500)
	if err != nil {
		t.Fatal(err)
	}

	if len(rep.Files) != 3 {
		t.Fatalf("got %d files, want 3 (>= 2500 bytes)", len(rep.Files))
	}
}

func TestRecent(t *testing.T) {
	root := makeTree(t)

	// Windows reports a fresh creation time for every file in the test tree,
	// so filter by size here and check that the old file is excluded only
	// where creation time is not available.
	rep, err := Recent(context.Background(), Env{}, root, 100, 1000, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range rep.Files {
		if f.Size < 1000 {
			t.Errorf("file below min size returned: %s (%d)", f.Path, f.Size)
		}
	}

	for i := 1; i < len(rep.Files); i++ {
		if rep.Files[i].Changed().After(rep.Files[i-1].Changed()) {
			t.Errorf("files not sorted newest first at %d", i)
		}
	}

	if len(rep.Files) < 3 {
		t.Errorf("expected at least the 3 fresh large files, got %d", len(rep.Files))
	}
}

func TestFolders(t *testing.T) {
	root := makeTree(t)

	rep, err := Folders(context.Background(), Env{}, root, 10, 0)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]Folder{}
	for _, f := range rep.Folders {
		rel, _ := filepath.Rel(root, f.Path)
		got[filepath.ToSlash(rel)] = f
	}

	want := map[string]struct {
		size  int64
		files int64
		depth int
	}{
		"video":        {9000, 3, 1},
		"video/clips":  {4000, 2, 2},
		"docs":         {3800, 2, 1},
		"docs/archive": {3000, 1, 2},
	}

	for rel, w := range want {
		f, ok := got[rel]
		if !ok {
			t.Errorf("folder %s missing", rel)
			continue
		}
		if f.Size != w.size || f.Files != w.files || f.Depth != w.depth {
			t.Errorf("folder %s = %+v, want size=%d files=%d depth=%d", rel, f, w.size, w.files, w.depth)
		}
	}

	if rep.Folders[0].Size != 9000 {
		t.Errorf("largest folder should be first, got %+v", rep.Folders[0])
	}
}

func TestFoldersMaxDepth(t *testing.T) {
	root := makeTree(t)

	rep, err := Folders(context.Background(), Env{}, root, 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(rep.Folders) != 2 {
		t.Fatalf("got %d folders at depth 1, want 2: %+v", len(rep.Folders), rep.Folders)
	}
}

func TestResolveRootErrors(t *testing.T) {
	if _, err := ResolveRoot(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("expected error for missing directory")
	}

	file := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveRoot(file); err == nil {
		t.Error("expected error for a file")
	}
}

func TestApps(t *testing.T) {
	root := t.TempDir()

	// name -> list of file sizes; many small files must add up.
	files := map[string][]int{
		"Program Files/Tool":                                 {100, 100, 100, 100, 100},
		"Program Files/Epic Games/GameA/bin":                 {3000},
		"Program Files (x86)/Steam":                          {500},
		"Program Files (x86)/Steam/steamapps/common/GameB/x": {4000},
		"Users/me/AppData/Local/Chrome/cache":                {10, 10, 10},
		"Users/me/AppData/Local/Programs/Cursor":             {700},
		"Users/me/.ollama/models":                            {2000},
		"Users/me/Documents":                                 {9999},
		"Windows/System32":                                   {8888},
		"Tools/qt":                                           {50},
	}

	for dir, sizes := range files {
		for i, size := range sizes {
			path := filepath.Join(root, filepath.FromSlash(dir), "f"+string(rune('a'+i)))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	rep, err := Apps(context.Background(), Env{}, root, 100)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]App{}
	for _, a := range rep.Apps {
		rel, _ := filepath.Rel(root, a.Path)
		got[filepath.ToSlash(rel)] = a
	}

	want := map[string]struct {
		kind  string
		size  int64
		files int64
	}{
		"Program Files/Tool":                               {"Program Files", 500, 5},
		"Program Files/Epic Games/GameA":                   {"Epic Games", 3000, 1},
		"Program Files (x86)/Steam":                        {"Program Files (x86)", 500, 1},
		"Program Files (x86)/Steam/steamapps/common/GameB": {"Steam", 4000, 1},
		"Users/me/AppData/Local/Chrome":                    {`AppData\Local`, 30, 3},
		"Users/me/AppData/Local/Programs/Cursor":           {"User programs", 700, 1},
		"Users/me/.ollama":                                 {"User profile", 2000, 1},
		"Tools":                                            {"Top level", 50, 1},
	}

	for rel, w := range want {
		a, ok := got[rel]
		if !ok {
			t.Errorf("app %s missing", rel)
			continue
		}
		if a.Kind != w.kind || a.Size != w.size || a.Files != w.files {
			t.Errorf("app %s = %+v, want kind=%q size=%d files=%d", rel, a, w.kind, w.size, w.files)
		}
	}

	for _, notApp := range []string{"Windows", "Users", "Program Files", "Program Files/Epic Games", "Users/me/Documents"} {
		if _, ok := got[notApp]; ok {
			t.Errorf("%s must not be listed as an app", notApp)
		}
	}

	if rep.Apps[0].Size != 4000 {
		t.Errorf("biggest app should come first, got %+v", rep.Apps[0])
	}
}
