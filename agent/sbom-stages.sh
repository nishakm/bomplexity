#!/usr/bin/env bash
# Component counts per stage for the Act 3 demo. Called by the Makefile:
#   sbom-stages.sh scorecard   table of component counts per stage
#   sbom-stages.sh diff        components only in the v2 images
# Image lists come from the manifests; SBOMs are made with syft and cached in
# sbom/ (keyed by image ID, so a rebuilt image is re-scanned).
set -euo pipefail
cd "$(dirname "$0")"

KIND_DIR=../kind
APP_URL=${APP_URL:-http://localhost:8080}
CACHE=sbom

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "SKIPPED: '$1' is not installed ($2)" >&2; return 1; }
}

images() { sed -nE 's/^[[:space:]]*image:[[:space:]]*([^[:space:]]+).*/\1/p' "$@" | sort -u; }
v1_images() { images "$KIND_DIR"/app.yaml "$KIND_DIR"/postgres.yaml "$KIND_DIR"/nginx.yaml; }
v2_only_images() { images agent.yaml; }

# image_sbom IMAGE -> path of a syft-json file (scanned if not cached)
image_sbom() {
  local id file
  id=$(docker image inspect -f '{{.Id}}' "$1" 2>/dev/null) || { echo "image $1 not found in docker" >&2; return 1; }
  file="$CACHE/$(echo "$1" | tr '/:' '__')-${id:7:12}.json"
  mkdir -p "$CACHE"
  [ -s "$file" ] || syft "docker:$1" -q -o syft-json > "$file"
  echo "$file"
}

count() { jq '.artifacts | length' "$1"; }

# components IMAGE -> "name version type" lines
components() { jq -r '.artifacts[] | "\(.name) \(.version) (\(.type))"' "$(image_sbom "$1")"; }

scorecard() {
  printf '%-34s %10s\n' "STAGE / ITEM" "COMPONENTS"
  printf '%-34s %10s\n' "----------------------------------" "----------"

  if need curl "needed to read the app's /sbom"; then
    if out=$(curl -fsS "$APP_URL/sbom" 2>/dev/null); then
      n=$(printf '%s\n' "$out" | tail -n +2 | grep -c . || true)
      printf '%-34s %10s\n' "1. v1 app /sbom (Go modules)" "$n"
    else
      echo "SKIPPED: could not read $APP_URL/sbom (is the demo up?)" >&2
    fi
  fi

  need syft "https://github.com/anchore/syft" || return 0
  need jq "needed to read syft output" || return 0
  need docker "needed to read the images" || return 0

  local total=0 n img
  for img in $(v1_images); do
    n=$(count "$(image_sbom "$img")") || continue
    printf '   %-31s %10s\n' "$img" "$n"; total=$((total + n))
  done
  printf '%-34s %10s\n' "2. v1 deployment images total" "$total"

  for img in $(v2_only_images); do
    n=$(count "$(image_sbom "$img")") || continue
    printf '   %-31s %10s\n' "$img" "$n"; total=$((total + n))
  done
  printf '%-34s %10s\n' "3. v2 deployment images total" "$total"
}

diff_v2() {
  need syft "https://github.com/anchore/syft" || return 0
  need jq "needed to read syft output" || return 0
  need docker "needed to read the images" || return 0
  local v1 v2 img
  v1=$(mktemp); v2=$(mktemp)
  trap 'rm -f "$v1" "$v2"' RETURN
  for img in $(v1_images); do components "$img"; done | LC_ALL=C sort -u > "$v1"
  for img in $(v2_only_images); do components "$img"; done | LC_ALL=C sort -u > "$v2"
  echo "Components in the v2 images that are not in v1 (name version (type)):"
  LC_ALL=C comm -13 "$v1" "$v2"
}

case "${1:-}" in
  scorecard) scorecard ;;
  diff) diff_v2 ;;
  *) echo "usage: $0 scorecard|diff" >&2; exit 2 ;;
esac
