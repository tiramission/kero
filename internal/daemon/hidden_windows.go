//go:build windows

package daemon

import (
	"path/filepath"
	"syscall"
)

// fileAttributeHidden is FILE_ATTRIBUTE_HIDDEN in Win32.
const fileAttributeHidden = 0x2

// entryHidden reports whether Windows hides dir/name by its file
// attributes. A dotfile is hidden by its name, which fsList checks too.
func entryHidden(dir, name string) bool {
	p, err := syscall.UTF16PtrFromString(filepath.Join(dir, name))
	if err != nil {
		return false
	}
	attrs, err := syscall.GetFileAttributes(p)
	return err == nil && attrs&fileAttributeHidden != 0
}
