//go:build !windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// Drives returns the filesystem root and the user's home directory.
func Drives() ([]Drive, error) {
	roots := []string{"/"}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, home)
	}

	var drives []Drive

	for _, root := range roots {
		d := Drive{Path: root, Kind: "Directory"}

		var st syscall.Statfs_t
		if err := syscall.Statfs(root, &st); err == nil {
			d.Total = uint64(st.Blocks) * uint64(st.Bsize)
			d.Free = uint64(st.Bavail) * uint64(st.Bsize)
		}

		drives = append(drives, d)
	}

	return drives, nil
}

// FileManager is the human-readable name of the system file manager.
var FileManager = map[bool]string{true: "Finder", false: "file manager"}[runtime.GOOS == "darwin"]

// Reveal shows the file in the system file manager.
func Reveal(path string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-R", path).Start()
	}
	return exec.Command("xdg-open", filepath.Dir(path)).Start()
}

// EnableColors reports whether stdout is a terminal.
func EnableColors() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
