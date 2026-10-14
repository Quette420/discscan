package finder

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gsaranov/discscan/internal/scan"
	"github.com/gsaranov/discscan/internal/top"
)

// Folder is a directory with the total size of everything beneath it.
type Folder struct {
	Path  string
	Size  int64 // bytes, including all subfolders
	Files int64 // file count, including all subfolders
	Depth int   // 1 = direct child of the scan root
}

// FolderReport is the result of the Folders mode.
type FolderReport struct {
	Summary
	Folders []Folder
}

type dirTotals struct {
	bytes, files int64
}

// Folders finds the `limit` biggest folders under root. maxDepth limits how
// deep below root a folder may be to be listed (0 = no limit); sizes always
// include the whole subtree.
func Folders(ctx context.Context, env Env, root string, limit, maxDepth int) (*FolderReport, error) {
	var (
		mu     sync.Mutex
		direct = make(map[string]dirTotals)
	)

	sum, err := walk(ctx, env, root, scan.Options{
		OnDir: func(dir string, bytes, files int64) {
			mu.Lock()
			direct[dir] = dirTotals{bytes, files}
			mu.Unlock()
		},
	})
	if err != nil {
		return nil, err
	}

	totals := aggregate(sum.Root, direct, nil)

	best := top.New(limit, func(a, b Folder) bool {
		return a.Size < b.Size
	})

	for dir, t := range totals {
		if dir == sum.Root {
			continue
		}

		depth := depthBelow(sum.Root, dir)
		if maxDepth > 0 && depth > maxDepth {
			continue
		}

		best.Add(Folder{Path: dir, Size: t.bytes, Files: t.files, Depth: depth})
	}

	return &FolderReport{Summary: sum, Folders: best.Sorted()}, nil
}

// aggregate turns per-directory direct sizes into recursive subtree sizes.
// Directories in stop keep their own total but do not add it to their
// parent, which lets nested apps be counted once.
func aggregate(root string, direct map[string]dirTotals, stop map[string]bool) map[string]dirTotals {
	totals := make(map[string]dirTotals, len(direct))
	dirs := make([]string, 0, len(direct))

	for dir, t := range direct {
		totals[dir] = t
		dirs = append(dirs, dir)
	}

	// A child's path is always longer than its parent's, so processing the
	// longest paths first pushes every subtree total up before it is read.
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})

	for _, dir := range dirs {
		if dir == root || stop[dir] {
			continue
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			continue
		}

		t, p := totals[dir], totals[parent]
		p.bytes += t.bytes
		p.files += t.files
		totals[parent] = p
	}

	return totals
}

func depthBelow(root, dir string) int {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
