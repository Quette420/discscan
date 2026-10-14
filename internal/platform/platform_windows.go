//go:build windows

package platform

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetLogicalDrives      = kernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW         = kernel32.NewProc("GetDriveTypeW")
	procGetDiskFreeSpaceExW   = kernel32.NewProc("GetDiskFreeSpaceExW")
	procGetVolumeInformationW = kernel32.NewProc("GetVolumeInformationW")
	procSetErrorMode          = kernel32.NewProc("SetErrorMode")
	procSetConsoleMode        = kernel32.NewProc("SetConsoleMode")
)

const (
	semFailCriticalErrors           = 0x0001
	enableVirtualTerminalProcessing = 0x0004
)

var driveKinds = map[uintptr]string{
	2: "Removable",
	3: "Fixed",
	4: "Network",
	5: "CD-ROM",
	6: "RAM disk",
}

// Drives lists ready drive letters with their capacity and label.
func Drives() ([]Drive, error) {
	// Suppress "There is no disk in the drive" system dialogs.
	procSetErrorMode.Call(semFailCriticalErrors)

	mask, _, err := procGetLogicalDrives.Call()
	if mask == 0 {
		return nil, err
	}

	var drives []Drive

	for i := range 26 {
		if mask&(1<<i) == 0 {
			continue
		}

		root := string(rune('A'+i)) + `:\`
		rootPtr, _ := syscall.UTF16PtrFromString(root)

		kind, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(rootPtr)))
		name, ok := driveKinds[kind]
		if !ok {
			continue
		}

		var freeToCaller, total, totalFree uint64

		r, _, _ := procGetDiskFreeSpaceExW.Call(
			uintptr(unsafe.Pointer(rootPtr)),
			uintptr(unsafe.Pointer(&freeToCaller)),
			uintptr(unsafe.Pointer(&total)),
			uintptr(unsafe.Pointer(&totalFree)),
		)
		if r == 0 {
			continue // not ready (empty card reader, disconnected share, ...)
		}

		var label [syscall.MAX_PATH + 1]uint16

		procGetVolumeInformationW.Call(
			uintptr(unsafe.Pointer(rootPtr)),
			uintptr(unsafe.Pointer(&label[0])),
			uintptr(len(label)),
			0, 0, 0, 0, 0,
		)

		drives = append(drives, Drive{
			Path:  root,
			Label: syscall.UTF16ToString(label[:]),
			Kind:  name,
			Total: total,
			Free:  totalFree,
		})
	}

	return drives, nil
}

// FileManager is the human-readable name of the system file manager.
const FileManager = "Explorer"

// Reveal opens Explorer with the given file or folder selected.
func Reveal(path string) error {
	cmd := exec.Command("explorer.exe")
	// explorer.exe needs the exact form /select,"path"; Go's default argument
	// quoting would wrap the whole switch in quotes, which explorer rejects.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: `explorer.exe /select,"` + path + `"`,
	}
	// explorer.exe returns a non-zero exit code even on success, so only
	// failures to start are reported.
	return cmd.Start()
}

// EnableColors turns on ANSI escape processing for the console and reports
// whether colored output can be used.
func EnableColors() bool {
	ok := true

	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		h := syscall.Handle(f.Fd())

		var mode uint32
		if err := syscall.GetConsoleMode(h, &mode); err != nil {
			ok = false // redirected to a file or pipe
			continue
		}

		r, _, _ := procSetConsoleMode.Call(uintptr(h), uintptr(mode|enableVirtualTerminalProcessing))
		if r == 0 {
			ok = false
		}
	}

	return ok
}
