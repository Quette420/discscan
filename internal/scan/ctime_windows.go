//go:build windows

package scan

import (
	"io/fs"
	"syscall"
	"time"
)

func createdTime(info fs.FileInfo) time.Time {
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.CreationTime.Nanoseconds())
	}
	return time.Time{}
}
