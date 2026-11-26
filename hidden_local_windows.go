//go:build windows

package main

import (
	"path/filepath"
	"syscall"
)

// localHidden reports whether a file in this machine's directory has
// Windows' hidden attribute. kerod marks it too, but an older kerod, still
// running after the app was updated, doesn't yet send that field: the app
// checks the local tree itself so Files hides the same entries meanwhile.
func localHidden(dir, name string) bool {
	attrs, err := syscall.GetFileAttributes(syscall.StringToUTF16Ptr(filepath.Join(dir, name)))
	return err == nil && attrs&syscall.FILE_ATTRIBUTE_HIDDEN != 0
}
