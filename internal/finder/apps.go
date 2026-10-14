package finder

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gsaranov/discscan/internal/scan"
	"github.com/gsaranov/discscan/internal/top"
)

// App is an installed program, game or per-app data folder with the total
// size of everything inside it, however small the individual files are.
type App struct {
	Path  string
	Kind  string // where it lives: "Program Files", "Steam", "AppData\Roaming", ...
	Size  int64  // bytes, excluding nested apps (Steam games are not counted in Steam)
	Files int64
}

// AppReport is the result of the Apps mode.
type AppReport struct {
	Summary
	Apps []App
}

// containers are folders whose direct children are individual apps or games.
// Matched against the end of a lower-case, slash-separated path.
var containers = []struct{ suffix, kind string }{
	{"/program files/windowsapps", "Store apps"},
	{"/program files", "Program Files"},
	{"/program files (x86)", "Program Files (x86)"},
	{"/programdata", "ProgramData"},
	{"/appdata/local/programs", "User programs"},
	{"/appdata/local/packages", "Store app data"},
	{"/appdata/local", `AppData\Local`},
	{"/appdata/roaming", `AppData\Roaming`},
	{"/appdata/locallow", `AppData\LocalLow`},
	{"/steamapps/common", "Steam"},
	{"/epic games", "Epic Games"},
	{"/gog games", "GOG"},
	{"/gog galaxy/games", "GOG"},
	{"/xboxgames", "Xbox"},
	{"/riot games", "Riot Games"},
	{"/ubisoft game launcher/games", "Ubisoft"},
	{"/ea games", "EA"},
	{"/games", "Games"},
}

// notApps are top-level folders of a drive that are not programs.
var notApps = map[string]bool{
	"windows":                   true,
	"users":                     true,
	"recovery":                  true,
	"perflogs":                  true,
	"boot":                      true,
	"config.msi":                true,
	"documents and settings":    true,
	"system volume information": true,
	"onedrivetemp":              true,
}

// Apps lists programs, games and app data folders under root, biggest first.
//
// An "app" is a direct child of a known container (Program Files, AppData,
// Steam/Epic/GOG libraries, ...), a top-level folder of the scanned path, or
// a dot-folder in a user profile (.ollama, .gradle, ...). Its size includes
// all files below it, so folders with thousands of small files show up too.
func Apps(ctx context.Context, env Env, root string, limit int) (*AppReport, error) {
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

	kinds := make(map[string]string)
	stop := make(map[string]bool)

	for dir := range direct {
		if kind := appKind(sum.Root, dir); kind != "" {
			kinds[dir] = kind
			stop[dir] = true
		}
	}

	totals := aggregate(sum.Root, direct, stop)

	best := top.New(limit, func(a, b App) bool {
		return a.Size < b.Size
	})

	for dir, kind := range kinds {
		t := totals[dir]
		best.Add(App{Path: dir, Kind: kind, Size: t.bytes, Files: t.files})
	}

	return &AppReport{Summary: sum, Apps: best.Sorted()}, nil
}

// appKind classifies dir as an app and returns its kind, or "" if it is not one.
func appKind(root, dir string) string {
	if dir == root || containerKind(dir) != "" {
		return ""
	}

	parent := filepath.Dir(dir)
	if kind := containerKind(parent); kind != "" {
		return kind
	}

	base := strings.ToLower(filepath.Base(dir))

	if parent == root {
		if notApps[base] || strings.HasPrefix(base, "$") {
			return ""
		}
		return "Top level"
	}

	if strings.HasPrefix(base, ".") && strings.EqualFold(filepath.Base(filepath.Dir(parent)), "users") {
		return "User profile"
	}

	return ""
}

func containerKind(dir string) string {
	p := strings.ToLower(filepath.ToSlash(dir))

	for _, c := range containers {
		if strings.HasSuffix(p, c.suffix) {
			return c.kind
		}
	}

	return ""
}
