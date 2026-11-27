# Releasing Kero

A release is a tag: pushing `vX.Y.Z` runs the Release workflow
(`.github/workflows/release.yml`), which builds the app for macOS, Windows
and Linux, the server build for each platform, runs the tests, takes its
notes from `CHANGELOG.fork.md`, and publishes the release once the assets
are checked. There is no draft left to publish by hand.

## Steps

1. Pick the version: a patch (2.1.4 → 2.1.5) for fixes and small
   changes, a minor for features.
2. Set it in `mygo.json` (`"version"`).
3. Add its section to the top of `CHANGELOG.fork.md`: what changed since
   the last release (`git log vX.Y.Z..HEAD`), for the people who use
   Kero, not for who reads the code. Name settings as the app shows them
   (Settings › Hosts), credit outside contributors (Thanks @name), and
   say when hosts get a change with their next update. Leave out what
   changes nothing a user would notice, and fixes not confirmed to work.
   The section's body is the release's notes: its heading, `## X.Y.Z`,
   is what the workflow finds.
4. Commit both as `Release X.Y.Z`, tag it `vX.Y.Z`, and push `main` and
   the tag:

   ```sh
   git commit -am "Release X.Y.Z"
   git tag vX.Y.Z
   git push origin main
   git push origin vX.Y.Z
   ```

5. Wait for the Release run of the tag to finish (about 6 minutes):

   ```sh
   gh run list --workflow release.yml --limit 1
   gh run watch <run id> --exit-status
   ```

   The workflow publishes the draft itself, without asking, after the
   assets job passes. If a job failed, fix it and rerun the tag's run; a
   tested release publishes when it succeeds.

   If only the publish job failed, run the Publish release workflow
   (`.github/workflows/publish-release.yml`) on `main`; it publishes the
   draft with the same notes from `CHANGELOG.fork.md`.

The tag must match `mygo.json`'s version, or the run stops at its first
job. Kero is released, so changes keep older apps, kerods and config
files working: `proto.Version` goes up only for a breaking change to the
protocol, not for a new optional field or frame.
