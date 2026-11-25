package remote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestParsePlatform(t *testing.T) {
	for _, c := range []struct {
		os, arch string
		goos     string
		goarch   string
	}{
		{"Linux", "x86_64", "linux", "amd64"},
		{"Darwin", "arm64", "darwin", "arm64"},
		{"MINGW64_NT-10.0", "AMD64", "windows", "amd64"},
		{"Linux", "aarch64", "linux", "arm64"},
	} {
		goos, goarch, err := ParsePlatform(c.os, c.arch)
		if err != nil || goos != c.goos || goarch != c.goarch {
			t.Errorf("ParsePlatform(%q, %q) = %s/%s, %v", c.os, c.arch, goos, goarch, err)
		}
	}
	for _, c := range []struct{ os, arch string }{
		{"Plan9", "amd64"},
		{"Linux", "riscv64"},
	} {
		if _, _, err := ParsePlatform(c.os, c.arch); err == nil {
			t.Errorf("ParsePlatform(%q, %q) accepted", c.os, c.arch)
		}
	}
}

func TestParseUnamesSkipsSSHWarnings(t *testing.T) {
	const out = "Warning: Permanently added 'devbox' (ED25519) to the list of known hosts.\nLinux\nx86_64\n"
	goos, goarch, err := parseUnames([]byte(out))
	if err != nil || goos != "linux" || goarch != "amd64" {
		t.Errorf("parseUnames with an SSH warning = %s/%s, %v", goos, goarch, err)
	}
	if _, _, err := parseUnames([]byte("Warning: Permanently added 'host' (ED25519) to the list of known hosts.\n")); err == nil {
		t.Error("parseUnames accepted only a warning")
	}
	goos, goarch, err = parseUnames([]byte("windows\nAMD64\n"))
	if err != nil || goos != "windows" || goarch != "amd64" {
		t.Errorf("parseUnames with PowerShell's two lines = %s/%s, %v", goos, goarch, err)
	}
}

func TestFileURL(t *testing.T) {
	for _, c := range []struct {
		goos, path, want string
	}{
		{"linux", "/tmp/kero install", "file:///tmp/kero%20install"},
		{"darwin", "/Users/me/Library/Caches/kero", "file:///Users/me/Library/Caches/kero"},
		{"windows", `C:\Users\me\AppData\Local\Temp\kero install`, "file:///C:/Users/me/AppData/Local/Temp/kero%20install"},
	} {
		if got := fileURL(c.goos, c.path); got != c.want {
			t.Errorf("fileURL(%q, %q) = %q, want %q", c.goos, c.path, got, c.want)
		}
	}
}

func TestRemoteJoin(t *testing.T) {
	if got := remoteJoin("linux", "/tmp/kero", "v2026.11.2", "install-server.sh"); got != "/tmp/kero/v2026.11.2/install-server.sh" {
		t.Errorf("linux join = %q", got)
	}
	if got := remoteJoin("windows", `C:\Temp\kero`, "v2026.11.2", "install-server.ps1"); got != `C:\Temp\kero\v2026.11.2\install-server.ps1` {
		t.Errorf("windows join = %q", got)
	}
}

func TestVerifySHA256(t *testing.T) {
	archive := []byte("kero server archive")
	sum := sha256.Sum256(archive)
	good := []byte(strings.ToUpper(hex.EncodeToString(sum[:])) + "  kero-server.tar.gz\n")
	if err := VerifySHA256(archive, good); err != nil {
		t.Errorf("VerifySHA256 = %v", err)
	}
	if err := VerifySHA256(archive, []byte("0000000000000000000000000000000000000000000000000000000000000000  kero-server.tar.gz\n")); err == nil {
		t.Error("VerifySHA256 accepted another sum")
	}
}

func TestInstallLocalRefuses(t *testing.T) {
	if _, err := InstallLocal(context.Background(), Addr{Scheme: "http", Host: "example.com"}, "2026.11.2", "linux", "amd64", true); err == nil {
		t.Error("InstallLocal accepted an http host")
	}
	if _, err := InstallLocal(context.Background(), Addr{Scheme: "ssh", Host: "devbox"}, "not-a-version", "linux", "amd64", true); err == nil {
		t.Error("InstallLocal accepted a bad version")
	}
}
