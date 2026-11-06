#!/usr/bin/env bash
#
# render-manual.sh — build a single manual.pdf from the walkthrough markdown.
#
# OS-independent by design: it renders with the pinned `pandoc/latex` Docker
# image so the output is byte-for-byte reproducible on macOS, Linux and Windows
# with no host TeX install. If Docker is unavailable it falls back to a locally
# installed pandoc (+ a LaTeX engine). The markdown and images are the single
# source of truth; this script only assembles them.
#
# Usage:
#   ./render-manual.sh                 # -> manual.pdf next to this script
#   ./render-manual.sh out.pdf         # custom output path
#   MANUAL_ENGINE=pandoc ./render-manual.sh   # force the host-pandoc path
#   MANUAL_ENGINE=docker ./render-manual.sh   # force the Docker path
#
set -euo pipefail

# Resolve the directory of this script so it works from any CWD.
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$HERE"

OUT="${1:-manual.pdf}"
IMAGE="pandoc/latex:3.8"          # pinned for reproducibility
ENGINE="${MANUAL_ENGINE:-auto}"   # auto | docker | pandoc

# The chapters, in order. README first (title/overview), then the numbered
# walkthroughs in lexical order. Add new files by dropping them in — the glob
# keeps numeric order.
TITLE="pt-plan — Walkthrough Manual"
FILES=(README.md)
for f in [0-9]*.md; do
  [ -e "$f" ] && FILES+=("$f")
done

# Shared pandoc arguments (identical for both engines so the output matches).
# DejaVu fonts cover the decorative glyphs the docs use (circled numbers, the
# role glyph, the sub-row arrow); the Docker image ships them. On the host
# fallback the font override is dropped (see below) so pandoc uses whatever is
# installed — the few decorative glyphs may degrade but the content is intact.
common_args=(
  --from=gfm
  --pdf-engine=xelatex
  --toc --toc-depth=2
  --number-sections
  --lua-filter=circled.lua
  -H header.tex
  -V documentclass=report
  -V geometry:margin=2cm
  -V linkcolor:blue
  -V colorlinks=true
  -V title="$TITLE"
  --resource-path=.
  -o "$OUT"
)

docker_available() {
  command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

render_docker() {
  echo "Rendering $OUT with Docker ($IMAGE) ..."
  # Mount this directory at /data (the image's default workdir) so relative
  # image paths (img/...) resolve. --user keeps the output file owned by us.
  docker run --rm \
    -v "$HERE:/data" \
    --user "$(id -u):$(id -g)" \
    "$IMAGE" \
    "${FILES[@]}" \
    "${common_args[@]}" \
    -V mainfont="DejaVu Sans" \
    -V monofont="DejaVu Sans Mono"
}

render_pandoc() {
  echo "Rendering $OUT with host pandoc ..."
  command -v pandoc >/dev/null 2>&1 || {
    echo "error: neither Docker nor pandoc is available." >&2
    echo "  install Docker (recommended) or 'brew install pandoc --cask basictex'." >&2
    exit 1
  }
  # Make a host TeX install (e.g. macOS BasicTeX in /Library/TeX/texbin) visible.
  export PATH="$PATH:/Library/TeX/texbin"
  # No forced mainfont here: use whatever the host provides so we don't depend
  # on a specific font being installed.
  pandoc "${FILES[@]}" "${common_args[@]}"
}

case "$ENGINE" in
  docker) render_docker ;;
  pandoc) render_pandoc ;;
  auto)
    if docker_available; then
      render_docker
    else
      echo "Docker not available; falling back to host pandoc." >&2
      render_pandoc
    fi
    ;;
  *) echo "error: MANUAL_ENGINE must be auto|docker|pandoc" >&2; exit 2 ;;
esac

echo "Wrote $HERE/$OUT"
