//go:build !windows

package daemon

// entryHidden is false where a name alone is what hides an entry.
func entryHidden(_, _ string) bool {
	return false
}
