//go:build !windows

package scan

import (
	"io/fs"
	"time"
)

// createdTime is not portable outside Windows; modification time is used instead.
func createdTime(fs.FileInfo) time.Time {
	return time.Time{}
}
