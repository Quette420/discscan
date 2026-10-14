// Package platform hides OS-specific details: drive enumeration, revealing
// files in the system file manager and console capabilities.
package platform

// Drive is a mounted volume that can be scanned.
type Drive struct {
	Path  string // scan root, e.g. `C:\` or `/`
	Label string // volume label, may be empty
	Kind  string // Fixed, Removable, Network, ...
	Total uint64 // capacity in bytes (0 if unknown)
	Free  uint64 // free space in bytes (0 if unknown)
}

// Used returns the occupied space in bytes.
func (d Drive) Used() uint64 {
	if d.Free > d.Total {
		return 0
	}
	return d.Total - d.Free
}
