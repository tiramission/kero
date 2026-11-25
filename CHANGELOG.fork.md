# Changelog

Fork releases of Kero. Upstream's changelog stays in `CHANGELOG.md`.

## 2026.11.5

- Files has an eye button in its header: it hides dotfiles and shows them
  again, and remembers the choice. It starts showing them, as Files has.
- A diff of a file under a Windows symbolic link or junction opens again.
  Reading a path no longer resolves its way out of the worktree, and a
  working file that Windows cannot read through the link falls back to the
  git patch; writes still refuse paths that lead outside the worktree.
- The macOS app remains omitted while GitHub's macOS runners are congested;
  the server packages for macOS and the other platforms are included.

## 2026.11.4

- Fix the platform probe of a Windows host over SSH: PowerShell now prints
  its OS and architecture as the two lines the probe reads, so installing
  with **Download on This Computer** works on Windows too.
- The macOS app remains omitted while GitHub's macOS runners are congested;
  the server packages for macOS and the other platforms are included.

## 2026.11.3

- Fix installing or updating a host over SSH when OpenSSH prints its
  known-host warning: the warning is no longer read as the host's platform,
  and no longer becomes part of a temporary path.
- `kero hosts ls` shows each host's **Local Download** state, and
  `kero hosts add --local-download` sets it.
- The macOS app remains omitted while GitHub's macOS runners are congested;
  the server packages for macOS and the other platforms are included.

## 2026.11.2

- Add **Settings › Hosts › Download on This Computer** (`local-download`):
  the app downloads a release here, verifies its signature and SHA-256, sends
  it to the host over SSH or WSL, and installs it with a `file://` mirror. A
  host that cannot reach GitHub can now be installed and updated. Off by
  default.
- `kero hosts install` and `kero -H <host> daemon update` follow the same
  setting, so the app and the command line install the same way.
- The macOS app remains omitted while GitHub's macOS runners are congested;
  the server packages for macOS and the other platforms are included.

## 2026.11.1

- First release from this fork under the `YYYY.MM.N` version rule, with the
  same features as upstream 2.1.36.
- The app and the server update from this repository's releases and verify
  them with the fork's own signing key.
- The server install scripts download from this fork's releases.
- No macOS app in this release: GitHub's macOS runners are congested. Linux
  and Windows builds, and the server packages for every platform, are
  included.
