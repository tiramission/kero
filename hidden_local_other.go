//go:build !windows

package main

// localHidden has no Windows attribute to read off Windows.
func localHidden(_, _ string) bool { return false }
