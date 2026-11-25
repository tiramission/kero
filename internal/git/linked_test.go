package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// A file under a linked directory, as a Windows junction or a POSIX
// symlink, is shown: reading a diff must not turn a path Windows cannot
// resolve through a link into a failure, while writes still refuse it.
func TestFileTextThroughLinkedDirectory(t *testing.T) {
	r := repo(t, map[string]string{"base.txt": "base\n"})
	outside := t.TempDir()
	write(t, filepath.Join(outside, "project-id"), "hello\n")
	link := filepath.Join(r.Root, ".freebuff")
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput(); err != nil {
			t.Fatalf("mklink: %v: %s", err, out)
		}
	} else if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	const path = ".freebuff/project-id"
	text, err := r.FileText(context.Background(), "HEAD", path, "")
	if err != nil {
		t.Fatalf("FileText: %v", err)
	}
	if text.New != "hello\n" || text.Binary || text.Large {
		t.Errorf("FileText is %+v", text)
	}
	changed, err := r.Changed(context.Background(), "HEAD", path)
	if err != nil || !changed {
		t.Errorf("Changed is %v, %v", changed, err)
	}
	if runtime.GOOS != "windows" {
		if _, err := r.local(path); err == nil {
			t.Error("a write path went through the link")
		}
	}
}
