package remote

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"kero/internal/shell"
	"kero/internal/update"
)

// InstallLocal downloads a release on this computer, checks it, and sends
// it to a host over SSH or WSL: the host needs no network of its own for
// the download. The install script runs from the copy sent with it, its
// KERO_RELEASES a file:// mirror of the same directory.
//
// version is the release to install, "" for the latest. A platform empty
// is found on the host. batch asks ssh nothing, as the app does.
func InstallLocal(ctx context.Context, a Addr, version, goos, goarch string, batch bool) (string, error) {
	switch a.Scheme {
	case "ssh", "wsl":
	default:
		return "", fmt.Errorf("Kero installs from this computer only over SSH or WSL, not %s", a.Scheme)
	}
	if a.Scheme == "wsl" && runtime.GOOS != "windows" {
		return "", fmt.Errorf("wsl:// addresses are of Windows")
	}
	if version == "" {
		v, err := update.Latest(ctx)
		if err != nil {
			return "", err
		}
		version = v
	}
	version = strings.TrimPrefix(version, "v")
	if !update.ValidVersion(version) {
		return "", fmt.Errorf("%q is not a version of Kero", version)
	}
	if goos == "" || goarch == "" {
		if a.Scheme == "ssh" {
			if err := FindShell(a, batch); err != nil {
				return "", err
			}
		}
		var err error
		goos, goarch, err = ProbePlatform(ctx, a, batch)
		if err != nil {
			return "", err
		}
	}
	if !slices.Contains(update.Platforms, goos+"/"+goarch) {
		return "", fmt.Errorf("the server build of Kero is not made for %s/%s", goos, goarch)
	}
	if goos == "windows" && a.Scheme == "ssh" && !Windows(a) {
		if err := FindShell(a, batch); err != nil {
			return "", err
		}
	}
	if goos == "windows" && a.Scheme == "ssh" && !Windows(a) {
		return "", fmt.Errorf("%s runs Windows without OpenSSH's PowerShell: install Kero there with its install script", a.Host)
	}

	name := update.Asset(version, goos, goarch)
	archive, err := update.Fetch(ctx, version, name)
	if err != nil {
		return "", fmt.Errorf("downloading Kero %s: %w", version, err)
	}
	sig, err := update.Fetch(ctx, version, name+".sig")
	if err != nil {
		return "", fmt.Errorf("downloading the signature of Kero %s: %w", version, err)
	}
	sha, err := update.Fetch(ctx, version, name+".sha256")
	if err != nil {
		return "", fmt.Errorf("downloading the SHA-256 of Kero %s: %w", version, err)
	}
	scriptName := "install-server.sh"
	if goos == "windows" {
		scriptName = "install-server.ps1"
	}
	script, err := update.Fetch(ctx, version, scriptName)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", scriptName, err)
	}
	if err := VerifySHA256(archive, sha); err != nil {
		return "", err
	}
	if err := update.Verify(archive, version, goos, goarch, string(sig)); err != nil {
		return "", err
	}

	temporary, err := makeRemoteTemp(ctx, a, goos, batch)
	if err != nil {
		return "", err
	}
	defer removeRemoteTemp(a, goos, temporary, batch)
	release := remoteJoin(goos, temporary, "v"+version)

	if err := makeRemoteDir(ctx, a, goos, release, batch); err != nil {
		return "", err
	}
	for _, f := range []struct {
		name string
		data []byte
	}{
		{scriptName, script},
		{name, archive},
		{name + ".sha256", sha},
	} {
		if err := pushRemoteFile(ctx, a, goos, release, f.name, f.data, batch); err != nil {
			return "", err
		}
	}
	return runInstallScript(ctx, a, goos, temporary, release, scriptName, version, batch)
}

// ProbePlatform finds the goos and goarch of a host, as its uname says,
// or its PowerShell.
func ProbePlatform(ctx context.Context, a Addr, batch bool) (string, string, error) {
	var out []byte
	var err error
	switch a.Scheme {
	case "ssh":
		if Windows(a) {
			script := `$a = $env:PROCESSOR_ARCHITECTURE; if ($env:PROCESSOR_ARCHITEW6432) { $a = $env:PROCESSOR_ARCHITEW6432 }; Write-Output ('windows/' + $a)`
			out, err = sshRun(ctx, a, batch, encodedPowerShell(script), nil)
		} else {
			out, err = sshRun(ctx, a, batch, "sh -c 'uname -s; uname -m'", nil)
		}
	case "wsl":
		if runtime.GOOS != "windows" {
			return "", "", fmt.Errorf("wsl:// addresses are of Windows")
		}
		out, err = wslRun(ctx, a, "uname -s; uname -m", nil)
	default:
		return "", "", fmt.Errorf("%s is not a host Kero installs on", a.Scheme)
	}
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return "", "", fmt.Errorf("the host did not say what it runs: %q", strings.TrimSpace(string(out)))
	}
	osName, arch, err := ParsePlatform(lines[0], lines[1])
	if err != nil {
		return "", "", err
	}
	return osName, arch, nil
}

// ParsePlatform reads the os and machine uname says, as a goos and goarch
// of the server build.
func ParsePlatform(osName, arch string) (string, string, error) {
	o := strings.ToLower(strings.TrimSpace(osName))
	switch {
	case strings.HasPrefix(o, "linux"):
		o = "linux"
	case strings.HasPrefix(o, "darwin"):
		o = "darwin"
	case strings.HasPrefix(o, "mingw"), strings.HasPrefix(o, "msys"), strings.HasPrefix(o, "cygwin"):
		o = "windows"
	case o == "windows":
		o = "windows"
	default:
		return "", "", fmt.Errorf("Kero's server build is not made for %s", strings.TrimSpace(osName))
	}
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "x86_64", "amd64":
		arch = "amd64"
	case "aarch64", "arm64":
		arch = "arm64"
	default:
		return "", "", fmt.Errorf("Kero's server build is not made for %s", strings.TrimSpace(arch))
	}
	if !slices.Contains(update.Platforms, o+"/"+arch) {
		return "", "", fmt.Errorf("Kero's server build is not made for %s/%s", o, arch)
	}
	return o, arch, nil
}

// VerifySHA256 checks an archive against the release's .sha256.
func VerifySHA256(archive, sha []byte) error {
	sum := sha256.Sum256(archive)
	want := strings.ToLower(strings.TrimSpace(string(sha)))
	if i := strings.IndexAny(want, " \t"); i >= 0 {
		want = want[:i]
	}
	if len(want) != sha256.Size*2 || want != hex.EncodeToString(sum[:]) {
		return fmt.Errorf("the archive's SHA-256 is not the release's")
	}
	return nil
}

func makeRemoteTemp(ctx context.Context, a Addr, goos string, batch bool) (string, error) {
	if goos == "windows" {
		suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
		script := `$d = Join-Path ([IO.Path]::GetTempPath()) 'kero-install-` + suffix + `'; New-Item -ItemType Directory -Force -Path $d | Out-Null; [Console]::Out.Write($d)`
		out, err := sshRun(ctx, a, batch, encodedPowerShell(script), nil)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(strings.TrimSpace(string(out)), `\/`), nil
	}
	var out []byte
	var err error
	if a.Scheme == "wsl" {
		out, err = wslRun(ctx, a, "mktemp -d", nil)
	} else {
		out, err = sshRun(ctx, a, batch, shScript("mktemp -d"), nil)
	}
	if err != nil {
		return "", err
	}
	d := strings.TrimSpace(string(out))
	if d == "" {
		return "", fmt.Errorf("the host made no temporary directory")
	}
	return d, nil
}

func removeRemoteTemp(a Addr, goos, dir string, batch bool) {
	if dir == "" {
		return
	}
	var script string
	if goos == "windows" {
		script = encodedPowerShell(`Remove-Item -LiteralPath ` + shell.PSQuote(dir) + ` -Recurse -Force -ErrorAction SilentlyContinue`)
	} else {
		script = "rm -rf " + shellQuote(dir)
	}
	if a.Scheme == "wsl" {
		_, _ = wslRun(context.Background(), a, script, nil)
		return
	}
	if goos != "windows" {
		script = shScript(script)
	}
	_, _ = sshRun(context.Background(), a, batch, script, nil)
}

func makeRemoteDir(ctx context.Context, a Addr, goos, dir string, batch bool) error {
	var script string
	if goos == "windows" {
		script = encodedPowerShell(`New-Item -ItemType Directory -Force -Path ` + shell.PSQuote(dir) + ` | Out-Null`)
	} else {
		script = "mkdir -p " + shellQuote(dir)
	}
	if a.Scheme == "wsl" {
		_, err := wslRun(ctx, a, script, nil)
		return err
	}
	if goos != "windows" {
		script = shScript(script)
	}
	_, err := sshRun(ctx, a, batch, script, nil)
	return err
}

func pushRemoteFile(ctx context.Context, a Addr, goos, dir, name string, data []byte, batch bool) error {
	path := remoteJoin(goos, dir, name)
	if goos == "windows" {
		encoded := base64.StdEncoding.EncodeToString(data)
		script := `$b = [Console]::In.ReadToEnd(); [IO.File]::WriteAllBytes(` + shell.PSQuote(path) + `, [Convert]::FromBase64String($b))`
		_, err := sshRun(ctx, a, batch, encodedPowerShell(script), []byte(encoded))
		return err
	}
	script := "cat > " + shellQuote(path)
	if a.Scheme == "wsl" {
		_, err := wslRun(ctx, a, script, data)
		return err
	}
	_, err := sshRun(ctx, a, batch, shScript(script), data)
	return err
}

func runInstallScript(ctx context.Context, a Addr, goos, temporary, release, scriptName, version string, batch bool) (string, error) {
	base := fileURL(goos, temporary)
	var script string
	if goos == "windows" {
		script = `$ErrorActionPreference = 'Stop'; $env:KERO_RELEASES = ` + shell.PSQuote(base) +
			`; $s = [IO.File]::ReadAllText(` + shell.PSQuote(remoteJoin(goos, release, scriptName)) + `); & ([scriptblock]::Create($s)) -Version ` + shell.PSQuote(version)
		script = encodedPowerShell(script)
	} else {
		script = "KERO_RELEASES=" + shellQuote(base) + " sh " + shellQuote(remoteJoin(goos, release, scriptName)) + " --version " + shellQuote(version)
		if a.Scheme == "ssh" {
			script = shScript(script)
		}
	}
	var out []byte
	var err error
	if a.Scheme == "wsl" {
		out, err = wslRun(ctx, a, script, nil)
	} else {
		out, err = sshRun(ctx, a, batch, script, nil)
	}
	return string(out), err
}

func remoteJoin(goos string, parts ...string) string {
	sep := "/"
	if goos == "windows" {
		sep = `\`
	}
	return strings.Join(parts, sep)
}

// shScript runs a POSIX script in sh, whatever shell the SSH server would
// otherwise run it with: fish and csh take no VAR=value command.
func shScript(script string) string {
	return "sh -c " + shellQuote(script)
}

func fileURL(goos, path string) string {
	if goos == "windows" {
		path = strings.ReplaceAll(path, `\`, "/")
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func sshRun(ctx context.Context, a Addr, batch bool, command string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, SSHCommand(), append(installSSH(a, batch), "--", a.Host, command)...)
	cmd.Env = sshEnv()
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, commandError(out, err)
	}
	return out, nil
}

func wslRun(ctx context.Context, a Addr, command string, stdin []byte) ([]byte, error) {
	args := []string{}
	if a.Host != "" {
		args = append(args, "-d", a.Host)
	}
	args = append(args, "--exec", "sh", "-c", command)
	cmd := exec.CommandContext(ctx, "wsl.exe", args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, commandError(out, err)
	}
	return out, nil
}

func commandError(out []byte, err error) error {
	msg := strings.TrimSpace(string(out))
	if i := strings.LastIndex(msg, "\n"); i >= 0 {
		msg = msg[i+1:]
	}
	if msg == "" {
		return err
	}
	return fmt.Errorf("%s: %s", msg, err)
}
