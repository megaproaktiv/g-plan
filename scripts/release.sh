#!/usr/bin/env bash
#
# release.sh - build the macOS app package of a pt-plan version and publish it as
# a GitHub release. One command for the whole procedure of documentation/ops/release.md.
#
#   scripts/release.sh                     build + publish the tag HEAD is on
#   scripts/release.sh --tag v1.2.3        same, for an explicit tag
#   scripts/release.sh --no-publish        RECREATE: build and verify only, nothing is uploaded
#   scripts/release.sh --republish         REPUBLISH: rebuild and replace the assets (and notes)
#                                          of a release that already exists
#   scripts/release.sh --smoke             also start the built app for a few seconds
#   scripts/release.sh --allow-dirty       do not refuse tracked, uncommitted changes
#   scripts/release.sh --no-windows        macOS only: skip the Windows zip
#
# What it does, in order:
#   1. checks: tag exists and is pushed, HEAD is exactly that tag, the working tree is clean,
#      CHANGELOG.md has a section for the version, and a GitHub login can write to the repo
#   2. syncs the app metadata (version, copyright, comments) into src/app/build/config.yml and
#      regenerates Info.plist; (re)builds the icon files via assets/icon/render-icon.sh if needed
#   3. builds the version-stamped UNIVERSAL binary (arm64 + amd64) with goreleaser
#   4. creates ptplan.app (ad-hoc signed) and ptplan.dmg with the project's own Wails tasks
#   5. verifies the result: architectures, version stamp, signature, icon files, dmg contents
#   5b. cross-compiles the Windows build (pure Go, no CGo) and zips pt-plan.exe with the readme
#      and the license; it is checked as a Windows GUI executable stamped with the version, but
#      it is NOT run (there is no Windows here): the notes say it is untested
#   6. stages dist/release-X.Y.Z/ with the dmg, the zipped app, the Windows zip, checksums.txt
#      and the notes (the notes are the CHANGELOG.md section of that version)
#   7. publishes (create, or with --republish replace) and verifies the upload by downloading
#      the assets again and comparing their checksums
#
# Requirements: macOS, git, goreleaser, wails3, gh, and the usual system tools (lipo, codesign,
# hdiutil, ditto, shasum, plutil). Icons need assets/icon/render-icon.sh (rsvg-convert, Pillow,
# and a full Xcode for the macOS 26 icon). The app is ad-hoc signed, NOT notarized.
#
# Environment overrides: APP_COPYRIGHT, APP_COMMENTS (shown in the app's Get Info).
set -euo pipefail

TAG=""; PUBLISH=1; REPUBLISH=0; ALLOW_DIRTY=0; SMOKE=0; WINDOWS=1
APP_COPYRIGHT="${APP_COPYRIGHT:-(c) $(date +%Y) pt-plan}"
APP_COMMENTS="${APP_COMMENTS:-Local graphical project-plan editor}"

log()  { printf '\n==> %s\n' "$*"; }
info() { printf '    %s\n' "$*"; }
die()  { printf '\nerror: %s\n' "$*" >&2; exit 1; }
usage() { sed -n '2,/^set -euo/p' "$0" | sed '$d' | sed 's/^# \{0,1\}//'; }

while [ $# -gt 0 ]; do
  case "$1" in
    --tag)         [ $# -ge 2 ] || die "--tag needs a value like v1.2.3"; TAG="$2"; shift 2 ;;
    --no-publish)  PUBLISH=0; shift ;;
    --republish)   REPUBLISH=1; shift ;;
    --allow-dirty) ALLOW_DIRTY=1; shift ;;
    --smoke)       SMOKE=1; shift ;;
    --no-windows)  WINDOWS=0; shift ;;
    -h|--help)     usage; exit 0 ;;
    *)             die "unknown option '$1' (see --help)" ;;
  esac
done
[ "$REPUBLISH" = 0 ] || [ "$PUBLISH" = 1 ] || die "--republish and --no-publish contradict each other"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
APP_DIR="$ROOT/src/app"
BUILD="$APP_DIR/build"
TMP="$(mktemp -d)"
MOUNT=""
cleanup() {
  [ -z "$MOUNT" ] || hdiutil detach "$MOUNT" -quiet >/dev/null 2>&1 || true
  rm -rf "$TMP"
}
trap cleanup EXIT

# quiet CMD...: run silently, but show the tail of the output if the command fails
quiet() {
  local out
  if ! out="$("$@" 2>&1)"; then
    printf '%s\n' "$out" | tail -25 >&2
    return 1
  fi
}

# ---------------------------------------------------------------- 1. checks
log "Checking the environment"
[ "$(uname)" = Darwin ] || die "this script builds the macOS app and must run on macOS"
need() { command -v "$1" >/dev/null 2>&1 || die "'$1' not found ($2)"; }
need git "install git"
need goreleaser "brew install goreleaser"
need wails3 "go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
for t in lipo codesign hdiutil ditto shasum plutil strings; do need "$t" "part of macOS / Xcode command line tools"; done
[ "$WINDOWS" = 0 ] || { need zip "part of macOS"; need file "part of macOS"; need go "brew install go"; }
[ "$PUBLISH" = 0 ] || need gh "brew install gh"
[ -d "$BUILD" ] || die "src/app/build is missing (it is git-ignored). Run 'wails3 task common:update:build-assets' in src/app first"

if [ -z "$TAG" ]; then
  TAG="$(git describe --tags --exact-match HEAD 2>/dev/null || true)"
  [ -n "$TAG" ] || die "HEAD is not on a tag. Tag it, or pass --tag vX.Y.Z (and 'git checkout vX.Y.Z')"
fi
case "$TAG" in v[0-9]*.[0-9]*.[0-9]*) ;; *) die "the tag must look like v1.2.3, got '$TAG'" ;; esac
VERSION="${TAG#v}"
git rev-parse -q --verify "refs/tags/$TAG" >/dev/null || die "tag $TAG does not exist locally"
[ "$(git rev-parse HEAD)" = "$(git rev-parse "$TAG^{commit}")" ] \
  || die "HEAD is not at $TAG, so the build would not match the release. Run: git checkout $TAG"
STRICT_PATHS="src assets scripts .goreleaser.yaml CHANGELOG.md"   # go into the binary, the icon or the notes
# shellcheck disable=SC2086
if [ "$ALLOW_DIRTY" = 0 ] && ! git diff --quiet HEAD -- $STRICT_PATHS; then
  git diff --name-only HEAD -- $STRICT_PATHS | sed 's/^/    /' >&2
  die "files that go into the build or the notes have uncommitted changes (commit them, or use --allow-dirty)"
fi
# other tracked changes (documentation, requirements, ...) cannot change the build: just mention them
others="$(git diff --name-only HEAD -- . ':(exclude)src' ':(exclude)assets' ':(exclude)scripts' \
  ':(exclude).goreleaser.yaml' ':(exclude)CHANGELOG.md' ':(exclude)memory.md' | tr '\n' ' ')"
[ -z "$others" ] || info "ignoring uncommitted changes outside the build: $others"
grep -q "^## \[$VERSION\]" CHANGELOG.md || die "CHANGELOG.md has no '## [$VERSION]' section (the release notes come from it)"
info "version $VERSION, tag $TAG, commit $(git rev-parse --short HEAD)"

REMOTE_URL="$(git remote get-url origin)"
REPO="$(printf '%s' "$REMOTE_URL" | sed -E 's#^.*github\.com[:/]##; s#\.git$##')"
[ -n "$REPO" ] || die "cannot work out the GitHub repository from origin ($REMOTE_URL)"

GH_STRIP_ENV=0
gh_() { if [ "$GH_STRIP_ENV" = 1 ]; then env -u GITHUB_TOKEN -u GH_TOKEN gh "$@"; else gh "$@"; fi; }
if [ "$PUBLISH" = 1 ]; then
  git ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1 \
    || die "tag $TAG is not pushed. Run: git push origin $TAG"
  # A GITHUB_TOKEN in the environment wins over the stored login. That token may be unable to see a
  # private repository (masked 404) or to create releases (403, a fine-grained token without
  # "contents: write"), so the stored login is tried FIRST and the environment token only if that
  # fails.
  if env -u GITHUB_TOKEN -u GH_TOKEN gh repo view "$REPO" --json name >/dev/null 2>&1; then
    GH_STRIP_ENV=1
    [ -z "${GITHUB_TOKEN:-}${GH_TOKEN:-}" ] || info "using the stored gh login (not the GITHUB_TOKEN of the environment)"
  elif gh repo view "$REPO" --json name >/dev/null 2>&1; then
    GH_STRIP_ENV=0
  else
    die "no GitHub login can access $REPO (try: gh auth login)"
  fi
  PERM="$(gh_ repo view "$REPO" --json viewerPermission --jq .viewerPermission)"
  case "$PERM" in WRITE|MAINTAIN|ADMIN) ;; *) die "your GitHub login has '$PERM' permission on $REPO, releases need WRITE" ;; esac
  if gh_ release view "$TAG" --repo "$REPO" >/dev/null 2>&1; then
    [ "$REPUBLISH" = 1 ] || die "release $TAG already exists. Use --republish to replace its assets"
    info "release $TAG exists: its assets and notes will be replaced"
  else
    [ "$REPUBLISH" = 0 ] || info "release $TAG does not exist yet: it will be created"
  fi
  info "publishing to $REPO as $PERM"
fi

# ---------------------------------------------------- 2. app metadata and icons
log "Preparing the app metadata and icons"
CFG="$BUILD/config.yml"
[ -f "$CFG" ] || die "$CFG is missing"
set_cfg() { # KEY VALUE: set an info: value in build/config.yml
  local v="${2//&/\\&}"
  sed -i '' -E "s|^(  $1: \")[^\"]*(\")|\1${v}\2|" "$CFG"
}
cfg_before="$(shasum "$CFG")"
set_cfg version "$VERSION"
set_cfg copyright "$APP_COPYRIGHT"
set_cfg comments "$APP_COMMENTS"
cfg_after="$(shasum "$CFG")"

ICONS="$ROOT/assets/icon"
if [ ! -f "$BUILD/darwin/icons.icns" ] || [ "$ICONS/appicon.svg" -nt "$BUILD/appicon.png" ] \
   || [ "$ICONS/layer.svg" -nt "$BUILD/appicon.png" ] \
   || { [ ! -f "$BUILD/darwin/Assets.car" ] && xcrun --find actool >/dev/null 2>&1; }; then
  info "building the icon files (assets/icon/render-icon.sh)"
  quiet "$ICONS/render-icon.sh" || die "render-icon.sh failed"
  cfg_after="icons-changed"
fi

plist_version="$(plutil -extract CFBundleShortVersionString raw -o - "$BUILD/darwin/Info.plist" 2>/dev/null || echo none)"
if [ "$cfg_before" != "$cfg_after" ] || [ "$plist_version" != "$VERSION" ]; then
  info "regenerating Info.plist for $VERSION"
  ( cd "$APP_DIR" && quiet wails3 task common:update:build-assets ) || die "update-build-assets failed"
fi
[ "$(plutil -extract CFBundleShortVersionString raw -o - "$BUILD/darwin/Info.plist")" = "$VERSION" ] \
  || die "Info.plist does not carry version $VERSION"

# ------------------------------------------------------------------ 3. binary
log "Building the universal binary with goreleaser"
# --skip=validate: the only tolerated dirty file is the local memory.md (checked above)
quiet goreleaser build --clean --skip=validate || die "goreleaser build failed"
BIN="$ROOT/dist/ptplan_darwin_all/pt-plan"
[ -f "$BIN" ] || die "goreleaser did not produce $BIN"
archs="$(lipo -archs "$BIN")"
case " $archs " in *" arm64 "*) ;; *) die "binary has no arm64 slice ($archs)" ;; esac
case " $archs " in *" x86_64 "*) ;; *) die "binary has no x86_64 slice ($archs)" ;; esac
strings "$BIN" > "$TMP/strings.txt"
grep -qx "$VERSION" "$TMP/strings.txt" || die "binary is not stamped with version $VERSION"
info "architectures: $archs, stamped $VERSION"

# --------------------------------------------------------------- 4. app + dmg
log "Creating ptplan.app and ptplan.dmg"
rm -rf "$APP_DIR/bin/ptplan.app" "$APP_DIR/bin/ptplan.dmg"
cp "$BIN" "$APP_DIR/bin/ptplan"
chmod +x "$APP_DIR/bin/ptplan"
# not 'darwin:package:dmg': it depends on 'package', which rebuilds a host-only binary over ours
( cd "$APP_DIR" && quiet wails3 task darwin:create:app:bundle && quiet wails3 task darwin:create:dmg ) \
  || die "creating the app bundle or the dmg failed"
# Never leave the universal (fat) binary at bin/ptplan. 'go build -o bin/ptplan' (which `task build`
# runs) refuses to overwrite a file that is not a Go-built object file and fails with
# "build output bin/ptplan already exists and is not an object file". The .app has its own copy.
rm -f "$APP_DIR/bin/ptplan"
APP="$APP_DIR/bin/ptplan.app"
DMG_BUILT="$APP_DIR/bin/ptplan.dmg"
[ -d "$APP" ] && [ -f "$DMG_BUILT" ] || die "app bundle or dmg missing after the build"

# ---------------------------------------------------------------- 5. verify
log "Verifying the package"
PL="$APP/Contents/Info.plist"
[ "$(plutil -extract CFBundleShortVersionString raw -o - "$PL")" = "$VERSION" ] || die "bundle version is not $VERSION"
exe="$(plutil -extract CFBundleExecutable raw -o - "$PL")"
[ -x "$APP/Contents/MacOS/$exe" ] || die "CFBundleExecutable '$exe' is not in the bundle (the app would not start)"
codesign --verify --deep --strict "$APP" || die "the app signature is not valid"
[ -f "$APP/Contents/Resources/icons.icns" ] || die "icons.icns is missing in the bundle"
if plutil -extract CFBundleIconName raw -o - "$PL" >/dev/null 2>&1; then
  [ -f "$APP/Contents/Resources/Assets.car" ] \
    || die "Info.plist names an icon (CFBundleIconName) but Assets.car is not in the bundle"
fi
info "bundle: version $VERSION, executable $exe, signature valid, icon files present"

hdiutil verify "$DMG_BUILT" >/dev/null 2>&1 || die "the dmg is damaged"
MOUNT="$(hdiutil attach -nobrowse -readonly -noautoopen "$DMG_BUILT" 2>/dev/null | awk -F'\t' '/\/Volumes\//{print $NF}' | tail -1)"
[ -n "$MOUNT" ] || die "could not mount the dmg"
[ -d "$MOUNT/ptplan.app" ] || die "the dmg does not contain ptplan.app"
[ -L "$MOUNT/Applications" ] || die "the dmg has no Applications shortcut"
codesign --verify --deep --strict "$MOUNT/ptplan.app" || die "the app inside the dmg is not validly signed"
[ "$(plutil -extract CFBundleShortVersionString raw -o - "$MOUNT/ptplan.app/Contents/Info.plist")" = "$VERSION" ] \
  || die "the app inside the dmg has the wrong version"
detached=0
for _ in 1 2 3 4 5; do
  if hdiutil detach "$MOUNT" -quiet >/dev/null 2>&1; then detached=1; break; fi
  sleep 2
done
[ "$detached" = 1 ] || die "could not unmount $MOUNT"
MOUNT=""
info "dmg: checksum valid, contains ptplan.app $VERSION and the Applications shortcut"

if [ "$SMOKE" = 1 ]; then
  "$APP/Contents/MacOS/$exe" > "$TMP/smoke.log" 2>&1 &
  pid=$!
  disown "$pid" 2>/dev/null || true   # no "Terminated" job message when we stop it
  sleep 5
  kill -0 "$pid" 2>/dev/null || { cat "$TMP/smoke.log" >&2; die "the app exited right after starting"; }
  kill "$pid" 2>/dev/null || true
  sleep 1
  kill -9 "$pid" 2>/dev/null || true
  grep -q "pt-plan $VERSION" "$TMP/smoke.log" || die "the app did not report version $VERSION at startup"
  info "smoke test: started and reported $VERSION"
fi

# ------------------------------------------------- 5b. Windows (cross-compiled)
WIN_ZIP="pt-plan_${VERSION}_windows_amd64.zip"
if [ "$WINDOWS" = 1 ]; then
  log "Cross-compiling the Windows build (not run here: untested)"
  # Wails on Windows is pure Go (no CGo), so macOS can build it. The .syso carries the icon, the
  # manifest and the version resource; it is removed again so it never lingers in the source tree.
  ( cd "$APP_DIR" && quiet wails3 task windows:generate:syso ARCH=amd64 ) || die "generating the Windows resources failed"
  WIN_DIR="$TMP/win/pt-plan_${VERSION}_windows_amd64"
  mkdir -p "$WIN_DIR"
  commit="$(git rev-parse --short HEAD)"; built="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  ( cd "$APP_DIR" && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 quiet go build -tags production -trimpath -buildvcs=false \
      -ldflags "-w -s -H windowsgui -X main.version=$VERSION -X main.commit=$commit -X main.date=$built" \
      -o "$WIN_DIR/pt-plan.exe" . ) || { rm -f "$APP_DIR"/*.syso; die "the Windows build failed"; }
  rm -f "$APP_DIR"/*.syso
  ftype="$(file -b "$WIN_DIR/pt-plan.exe")"
  case "$ftype" in *"PE32+"*"GUI"*"x86-64"*) ;; *) die "pt-plan.exe is not a 64-bit Windows GUI executable ($ftype)" ;; esac
  strings "$WIN_DIR/pt-plan.exe" > "$TMP/winstrings.txt"
  grep -qx "$VERSION" "$TMP/winstrings.txt" || die "pt-plan.exe is not stamped with version $VERSION"
  cp readme.md LICENSE "$WIN_DIR/"
  printf 'pt-plan %s for Windows (x64)\r\n\r\nThis build has NOT been tested on Windows yet.\r\nRun pt-plan.exe. Windows needs the WebView2 runtime (part of Windows 11 and of current\r\nMicrosoft Edge); SmartScreen may warn because the file is not signed.\r\n' "$VERSION" > "$WIN_DIR/README-windows.txt"
  info "pt-plan.exe: $ftype, stamped $VERSION"
fi

# ----------------------------------------------------------------- 6. stage
log "Staging the release files"
OUT="$ROOT/dist/release-$VERSION"
DMG="pt-plan_${VERSION}_macos_universal.dmg"
ZIP="pt-plan_${VERSION}_macos_universal.app.zip"
rm -rf "$OUT"; mkdir -p "$OUT"
cp "$DMG_BUILT" "$OUT/$DMG"
ditto -c -k --keepParent "$APP" "$OUT/$ZIP"
if [ "$WINDOWS" = 1 ]; then
  ( cd "$TMP/win" && rm -f "$OUT/$WIN_ZIP" && zip -qr "$OUT/$WIN_ZIP" "pt-plan_${VERSION}_windows_amd64" )
  unzip -l "$OUT/$WIN_ZIP" > "$TMP/winzip.txt"   # (a pipe into grep -q would fail with SIGPIPE under pipefail)
  grep -q "pt-plan.exe" "$TMP/winzip.txt" || die "the Windows zip does not contain pt-plan.exe"
  info "$(ls -l "$OUT/$WIN_ZIP" | awk '{print $5}') bytes  $WIN_ZIP"
fi
FILES=("$DMG" "$ZIP"); [ "$WINDOWS" = 0 ] || FILES+=("$WIN_ZIP")
( cd "$OUT" && shasum -a 256 "${FILES[@]}" > checksums.txt )
# the zip must still hold a valid, universal app
ditto -x -k "$OUT/$ZIP" "$TMP/unzip"
codesign --verify --deep --strict "$TMP/unzip/ptplan.app" || die "the app in the zip is not validly signed"
info "$(ls -l "$OUT/$DMG" | awk '{print $5}') bytes  $DMG"
info "$(ls -l "$OUT/$ZIP" | awk '{print $5}') bytes  $ZIP"

NOTES="$OUT/release-notes.md"
{
  printf '## pt-plan %s\n\n' "$TAG"
  printf 'Desktop Gantt / project-plan editor (Wails v3).\n\n'
  printf '### Download (macOS, universal: Apple Silicon + Intel)\n\n'
  printf '| File | Use |\n|------|-----|\n'
  printf '| `%s` | Open it and drag **ptplan** onto **Applications** |\n' "$DMG"
  printf '| `%s` | The same app as a plain zip |\n' "$ZIP"
  printf '| `checksums.txt` | SHA-256 of all files |\n\n'
  if [ "$WINDOWS" = 1 ]; then
    printf '### Windows (x64) - NOT TESTED\n\n'
    printf '`%s` holds `pt-plan.exe`. It is cross-compiled from macOS and has **not been run on Windows\n' "$WIN_ZIP"
    printf 'yet**; expect problems and please report them. It needs the WebView2 runtime (part of\n'
    printf 'Windows 11 and current Edge); SmartScreen may warn because the file is not signed.\n\n'
  fi
  printf 'The macOS app is **ad-hoc signed, not notarized**. When you first open a downloaded copy,\n'
  printf 'macOS may refuse it ("Apple cannot check it for malicious software"). Either open\n'
  printf '**System Settings > Privacy & Security** and choose **Open Anyway**, or remove the\n'
  printf 'quarantine flag once:\n\n'
  printf '```sh\nxattr -dr com.apple.quarantine /Applications/ptplan.app\n```\n\n---\n\n'
  awk -v v="$VERSION" '$0 ~ "^## \\[" v "\\]" {f=1; next} /^## \[/ {if (f) exit} f' CHANGELOG.md
} > "$NOTES"

if [ "$PUBLISH" = 0 ]; then
  log "Done (not published): the files are in $OUT"
  ls -1 "$OUT" | sed 's/^/    /'
  exit 0
fi

# --------------------------------------------------------------- 7. publish
log "Publishing $TAG to $REPO"
UPLOAD=()
for f in "${FILES[@]}" checksums.txt; do UPLOAD+=("$OUT/$f"); done
if gh_ release view "$TAG" --repo "$REPO" >/dev/null 2>&1; then
  gh_ release upload "$TAG" --repo "$REPO" --clobber "${UPLOAD[@]}" >/dev/null
  gh_ release edit "$TAG" --repo "$REPO" --title "pt-plan $TAG" --notes-file "$NOTES" >/dev/null
  info "replaced the assets and notes of the existing release"
else
  gh_ release create "$TAG" --repo "$REPO" --verify-tag --title "pt-plan $TAG" \
    --notes-file "$NOTES" "${UPLOAD[@]}" >/dev/null
  info "created the release"
fi

log "Verifying the published release"
state="$(gh_ release view "$TAG" --repo "$REPO" --json isDraft,isPrerelease,url \
  --jq '"draft=\(.isDraft) prerelease=\(.isPrerelease) \(.url)"')"
info "$state"
mkdir -p "$TMP/dl"
gh_ release download "$TAG" --repo "$REPO" --pattern 'pt-plan_*' --dir "$TMP/dl" >/dev/null
( cd "$TMP/dl" && shasum -a 256 "${FILES[@]}" ) | diff - "$OUT/checksums.txt" >/dev/null \
  || die "the downloaded assets do not match checksums.txt"
for f in "${FILES[@]}" checksums.txt; do
  want="$(wc -c < "$OUT/$f" | tr -d ' ')"
  got="$(gh_ release view "$TAG" --repo "$REPO" --json assets --jq ".assets[] | select(.name==\"$f\") | .size")"
  [ "$want" = "$got" ] || die "asset $f has $got bytes on GitHub, expected $want"
done
info "downloaded the assets again: checksums and sizes match"
case "$state" in draft=true*) info "NOTE: the release is still a draft; publish it on GitHub" ;; esac
log "Done: ${state##* }"
