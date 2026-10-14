// Package hints recognizes well-known space hogs (page file, crash dumps,
// caches, leftover installers, ...) and says what can be done about them.
package hints

import (
	"path/filepath"
	"slices"
	"strings"
)

// Action is the suggested way to deal with an item.
type Action int

const (
	Delete  Action = iota + 1 // safe to delete
	Cache                     // cache, regenerated or re-downloaded on demand
	Move                      // personal data, move to another drive
	System                    // managed by Windows, change via settings
	Careful                   // holds app data, clean only from inside the app
)

// Label is a short tag for tables.
func (a Action) Label() string {
	switch a {
	case Delete:
		return "delete"
	case Cache:
		return "cache"
	case Move:
		return "move"
	case System:
		return "system"
	case Careful:
		return "careful"
	}
	return ""
}

// Reclaimable reports whether the space can be freed without losing data.
func (a Action) Reclaimable() bool {
	return a == Delete || a == Cache
}

// Hint is advice about a file or folder.
type Hint struct {
	Action Action
	Note   string
}

// item is a normalized path: "/c:/users/me/file.zip" (+ "/" for folders).
type item struct {
	path string
	base string
	ext  string
}

func (it item) has(parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(it.path, p) {
			return true
		}
	}
	return false
}

func (it item) extIn(exts ...string) bool {
	return slices.Contains(exts, it.ext)
}

type rule struct {
	match func(item) bool
	hint  Hint
}

var (
	archiveExts   = []string{".zip", ".rar", ".7z", ".iso", ".tar", ".gz", ".xz", ".img"}
	installerExts = []string{".exe", ".msi", ".msix", ".appx", ".dmg", ".pkg"}
	videoExts     = []string{".mkv", ".avi", ".mp4", ".mov", ".wmv", ".m4v", ".ts", ".webm"}
	diskImageExts = []string{".vhd", ".vhdx", ".vmdk", ".vdi", ".qcow2"}
)

// rules are checked in order; the first match wins.
var rules = []rule{
	// Windows system files.
	{func(it item) bool { return it.base == "pagefile.sys" }, Hint{System,
		"page file: move to another drive or shrink in System Properties → Advanced → Performance → Virtual memory"}},
	{func(it item) bool { return it.base == "hiberfil.sys" }, Hint{System,
		"hibernation file: 'powercfg /h off' (admin) removes it, '/h /type reduced' halves it"}},
	{func(it item) bool { return it.base == "swapfile.sys" }, Hint{System,
		"swap file for Store apps, managed together with the page file"}},
	{func(it item) bool { return it.has("/windows/winsxs/") }, Hint{System,
		"component store: clean with 'Dism /Online /Cleanup-Image /StartComponentCleanup'"}},
	{func(it item) bool { return it.has("/windows/installer/") }, Hint{Careful,
		"MSI cache needed to repair/uninstall programs, never delete by hand"}},
	{func(it item) bool { return it.has("/windows.old/") }, Hint{Delete,
		"previous Windows installation: remove with Disk Cleanup → Clean up system files"}},
	{func(it item) bool { return it.has("/windows/softwaredistribution/download/") }, Hint{Cache,
		"Windows Update download cache: Disk Cleanup → Windows Update Cleanup"}},
	{func(it item) bool { return it.has("/$recycle.bin/") }, Hint{Delete,
		"in the Recycle Bin: empty it"}},
	{func(it item) bool { return it.has("/system volume information/") }, Hint{System,
		"restore points: limit in System Protection settings"}},

	// Crash and memory dumps.
	{func(it item) bool { return it.extIn(".dmp", ".mdmp") || it.has("/minidump/", "/crashdumps/") }, Hint{Delete,
		"crash dump, only useful for debugging that crash"}},
	{func(it item) bool { return it.ext == ".hprof" }, Hint{Delete,
		"Java heap dump written after an IDE/JVM crash"}},

	// Unfinished downloads and temp files.
	{func(it item) bool {
		return it.extIn(".crdownload", ".part", ".partial", ".!ut", ".!qb") || strings.HasPrefix(it.base, "~utorrentpartfile")
	}, Hint{Delete, "unfinished download"}},
	{func(it item) bool { return it.has("/appdata/local/temp/", "/windows/temp/", "/tmp/") }, Hint{Delete,
		"temporary file: safe to delete when the program that made it is closed"}},
	{func(it item) bool { return it.extIn(".tmp", ".temp") }, Hint{Delete, "temporary file"}},

	// Developer tools.
	{func(it item) bool { return it.has("/.ollama/models/") }, Hint{Careful,
		"Ollama model: remove with 'ollama rm <name>' or relocate with OLLAMA_MODELS"}},
	{func(it item) bool { return it.has("/ollama/updates") }, Hint{Delete, "downloaded Ollama update installer"}},
	{func(it item) bool { return it.has("/deriveddatacache/") }, Hint{Cache,
		"Unreal Engine derived data cache, rebuilt automatically"}},
	{func(it item) bool { return it.has("/epicgameslauncher/vaultcache/") }, Hint{Cache,
		"Epic Launcher vault cache (Fab/Marketplace downloads), re-downloaded when needed"}},
	{func(it item) bool {
		return it.has("/.gradle/caches/", "/.m2/repository/", "/go/pkg/mod/", "/.nuget/packages/",
			"/npm-cache/", "/pip/cache/", "/yarn/cache/", "/pnpm/store/", "/.cargo/registry/", "/.cache/")
	}, Hint{Cache, "package cache, re-downloaded on demand"}},
	{func(it item) bool { return it.has("/node_modules/") }, Hint{Cache,
		"node_modules: restored by 'npm install'"}},
	{func(it item) bool { return it.has("/jetbrains/") && it.has("/caches/", "/index/", "/log/") }, Hint{Cache,
		"JetBrains IDE caches/indexes: File → Invalidate Caches, or delete for old IDE versions"}},
	{func(it item) bool { return it.has("/appdata/local/jetbrains/") }, Hint{Cache,
		"JetBrains caches/indexes: delete folders of IDE versions you no longer use"}},
	{func(it item) bool { return it.has("/appdata/local/unrealengine/") }, Hint{Cache,
		"Unreal Engine shared DDC/Zen cache, rebuilt automatically (first project open gets slower)"}},
	{func(it item) bool {
		return it.has("/appdata/local/uv/", "/appdata/local/pip/", "/appdata/local/go-build/")
	}, Hint{Cache,
		"package/build cache, re-downloaded on demand ('uv cache clean', 'go clean -cache')"}},
	{func(it item) bool { return it.has("/programdata/package cache/") }, Hint{Careful,
		"installer cache used to repair/uninstall Visual Studio and runtimes"}},
	{func(it item) bool { return it.has("/appdata/roaming/telegram desktop/") }, Hint{Careful,
		"Telegram media cache: Settings → Advanced → Manage local storage"}},
	{func(it item) bool { return it.base == "state.vscdb" }, Hint{Careful,
		"VS Code/Cursor state database (settings, AI chat history): don't delete while you need them"}},

	// Browsers and apps.
	{func(it item) bool {
		return it.has("/user data/") && it.has("/file system/", "/indexeddb/", "/service worker/")
	}, Hint{Careful,
		"website storage: clear per site in chrome://settings/content/all (sort by data)"}},
	{func(it item) bool {
		return it.has("/cache/", "/cache_data/", "/code cache/", "/gpucache/", "/shadercache/", "/dxcache/")
	}, Hint{Cache,
		"application cache, cleared from the app or rebuilt automatically"}},

	// Personal files.
	{func(it item) bool { return it.has("/downloads/") && it.extIn(installerExts...) }, Hint{Delete,
		"installer in Downloads: not needed once the program is installed"}},
	{func(it item) bool { return it.has("/downloads/") && it.extIn(archiveExts...) }, Hint{Delete,
		"archive in Downloads: delete once unpacked or installed, or move to another drive"}},
	{func(it item) bool { return it.extIn(videoExts...) }, Hint{Move, "video: move to a data drive"}},
	{func(it item) bool { return it.extIn(diskImageExts...) }, Hint{Careful,
		"virtual machine disk: compact or move it from the VM software"}},
	{func(it item) bool { return it.ext == ".log" }, Hint{Delete, "log file"}},
}

// ForFile returns advice for a file, if any.
func ForFile(path string) (Hint, bool) {
	return lookup(newItem(path, false))
}

// ForDir returns advice for a folder, if any.
func ForDir(path string) (Hint, bool) {
	return lookup(newItem(path, true))
}

func newItem(path string, dir bool) item {
	p := "/" + strings.TrimPrefix(strings.ToLower(filepath.ToSlash(path)), "/")
	if dir && !strings.HasSuffix(p, "/") {
		p += "/"
	}

	base := strings.ToLower(filepath.Base(path))

	ext := ""
	if !dir {
		ext = filepath.Ext(base)
	}

	return item{path: p, base: base, ext: ext}
}

func lookup(it item) (Hint, bool) {
	for _, r := range rules {
		if r.match(it) {
			return r.hint, true
		}
	}
	return Hint{}, false
}
