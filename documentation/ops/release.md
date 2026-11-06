# Releasing pt-plan

pt-plan is a [Wails v3](https://v3.wails.io) desktop app. The Go main package
(module `ptplan`) lives in `src/app` and embeds `src/app/frontend/dist`. The
macOS webview uses CGO, so the macOS app is built on a Mac. The Windows build is pure Go
(no CGo) and is cross-compiled from macOS by `scripts/release.sh`; it is **untested**.

All commands below run from `src/app` unless noted.

## 0. The short way: `scripts/release.sh`

After step 1 below (changelog, commit, tag, push), one command builds the macOS
app package of that tag and publishes it as a GitHub release:

```sh
git checkout vX.Y.Z            # the script builds exactly the tag, and checks that HEAD is on it
scripts/release.sh             # build, verify, publish
```

| Command | What it does |
|---------|--------------|
| `scripts/release.sh` | Build + verify + **create** the release for the tag HEAD is on |
| `scripts/release.sh --tag vX.Y.Z` | Same, for an explicit tag (HEAD must be at it) |
| `scripts/release.sh --no-publish` | **Recreate** the package only; nothing is uploaded. The files land in `dist/release-X.Y.Z/` |
| `scripts/release.sh --republish` | **Republish**: rebuild and replace the assets and notes of a release that already exists |
| `scripts/release.sh --smoke` | Also start the built app for a few seconds |
| `scripts/release.sh --no-windows` | macOS only: skip the Windows zip |
| `scripts/release.sh --help` | Full description |

It refuses to run when the tag is not pushed, HEAD is not the tag, files that go
into the build (`src/`, `assets/`, `scripts/`, `.goreleaser.yaml`,
`CHANGELOG.md`) have uncommitted changes, or `CHANGELOG.md` has no section for
the version (that section becomes the release notes). It builds the stamped
universal binary (arm64 + amd64) with goreleaser, bundles `ptplan.app` and
`ptplan.dmg` with the Wails tasks, verifies architectures, version stamp,
signature, icon files and the dmg contents, and after uploading it downloads the
assets again to compare checksums. Unless `--no-windows` is given it also
cross-compiles `pt-plan.exe` (Windows x64, with icon, manifest and version resource),
checks it is a 64-bit GUI executable stamped with the version, and ships it as
`pt-plan_<version>_windows_amd64.zip`. It cannot run it, so the release notes mark
the Windows build as not tested.

Things worth knowing:

- **GitHub login.** A `GITHUB_TOKEN` in the environment takes precedence over the
  stored `gh` login; for a private repository a token without access produces only
  a masked 404. The script tests the environment login first and falls back to the
  stored login by itself.
- **Not notarized.** The app is ad-hoc signed. The release notes tell users how to
  open it (Privacy & Security > Open Anyway, or `xattr -dr com.apple.quarantine`).
- **Rebuilds are not byte-identical** (signatures and the dmg contain timestamps),
  so `--republish` changes the checksums; `checksums.txt` is replaced with the
  assets.
- **App metadata.** Version, copyright and comments are written into the git-ignored
  `src/app/build/config.yml` by the script. Override the text with
  `APP_COPYRIGHT` and `APP_COMMENTS`.
- **Icons** come from `assets/icon` (see section 4).

The sections below describe the individual steps the script automates.

## 1. Cut a version

1. Move the `[Unreleased]` notes in `CHANGELOG.md` under a new
   `## [X.Y.Z] - YYYY-MM-DD` heading.
2. Commit, then tag and push:
   ```sh
   git commit -am "Release vX.Y.Z: ..."
   git tag -a vX.Y.Z -m "pt-plan vX.Y.Z"
   git push origin main
   git push origin vX.Y.Z
   ```
   `main.go` reads `version`/`commit`/`date` from `-ldflags` at build time, so
   the running app logs the version it was built as.

## 2. GitHub release with goreleaser (raw universal binary)

`.goreleaser.yaml` (repo root) builds the frontend + Wails bindings in its
`before` hooks, compiles a CGO universal macOS binary (arm64 + amd64), archives
it, and attaches it to a GitHub release.

```sh
# from the repo root
goreleaser check                       # validate the config
goreleaser release --snapshot --clean --skip=publish   # local dry run, no tag needed
GITHUB_TOKEN=<token> goreleaser release --clean         # real release at the current tag
```

The token must be a **github.com** token with `contents: write` on
`megaproaktiv/g-plan`. Do not paste it on a command line that is logged; export
it in your shell (`export GITHUB_TOKEN=...`) or use a credential store. If the
publish step 404s with an otherwise-valid token, the token lacks release
permission (GitHub masks 403 as 404 on the releases endpoint).

Artifacts land in `dist/` (`pt-plan_X.Y.Z_darwin_all.tar.gz`, `checksums.txt`).

## 3. macOS .app / .dmg (optional, local)

goreleaser ships only the raw binary. To build a double-clickable app:

```sh
wails3 package                         # bin/ptplan.app  (AD-HOC signed only)
wails3 task darwin:package:dmg         # bin/ptplan.dmg
```

`wails3 package` ad-hoc signs the bundle (`codesign --sign -`). An ad-hoc signed
app runs on the building machine but is NOT distributable without warnings.

For distribution you need an Apple Developer ID and a one-time `wails3 setup`:

```sh
wails3 setup                           # store signing identity + notary profile
wails3 task darwin:sign                # Developer ID sign the .app
wails3 task darwin:sign:notarize       # sign AND notarize (needs the notary profile)
```

Attach the signed/notarized `.app` or `.dmg` to the GitHub release manually
(`gh release upload vX.Y.Z bin/ptplan.dmg`) if you want to ship it alongside the
binary archive.

## 4. The application icon

The icon sources live in `assets/icon/` (the Wails build folder `src/app/build/`
is git-ignored, so the icon must not live only there). After changing
`appicon.svg` or `layer.svg`, run:

```sh
assets/icon/render-icon.sh
```

It writes the PNG, `icons.icns` (all sizes up to 1024 px), the Windows `.ico`,
the `.dmg` file icon and, **if a full Xcode is installed**, `Assets.car` (the
macOS 26 Icon Composer icon, compiled by `actool`). Without Xcode it removes any
stale `Assets.car` and warns, and the app falls back to `icons.icns`. Xcode
needs its license accepted once (`sudo xcodebuild -license accept`). Afterwards
run `wails3 task common:update:build-assets` so `Info.plist` matches the files.

`task build` and `task package` run `wails3 generate icons` whenever the icon
sources changed. With Xcode installed that step replaces `icons.icns` by a reduced
set (nothing above 256 px). If you package by hand afterwards, run
`assets/icon/render-icon.sh` again; `scripts/release.sh` is not affected.

Also note: `go build -o bin/ptplan` (what `task build` runs) refuses to overwrite
a universal (lipo) binary. If you build a universal binary by hand, delete
`src/app/bin/ptplan` afterwards; `scripts/release.sh` does this itself.

## Notes

- `ld: warning: object file ... was built for newer 'macOS' version` lines
  during the build are harmless SDK-version mismatch noise.
- The generated Wails binding `*.js` files are not tracked (only the `*.ts`);
  vite/tsc compile them. The `before` hooks regenerate them for the release.
