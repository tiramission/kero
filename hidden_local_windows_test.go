//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLocalHiddenWindows(t *testing.T) {
	dir := t.TempDir()
	name := "hidden.txt"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("hidden"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("cmd", "/c", "attrib", "+h", path).CombinedOutput(); err != nil {
		t.Fatalf("attrib +h: %v: %s", err, out)
	}
	if !localHidden(dir, name) {
		t.Fatal("a file Windows hides did not read as hidden")
	}
	if localHidden(dir, "."+name) {
		t.Fatal("a missing file read as hidden")
	}
}
