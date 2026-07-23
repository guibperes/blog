#!/usr/bin/env bash
#
# publish.sh - publish a Hugo draft:
#   1. update the front matter `date` field to the current timestamp
#   2. replace `draft: true` with `draft: false` (if present)
#   3. rename YYYY-MM-DD-slug.md using today's date
#
# Usage: ./publish.sh content/posts/2026-03-30-montanha-ascese.md
#        ./publish.sh -n file.md    # dry run: only print what would happen

set -euo pipefail

DRY_RUN=0
[[ "${1:-}" == "-n" ]] && {
  DRY_RUN=1
  shift
}

FILE="${1:-}"
[[ -n "$FILE" && -f "$FILE" ]] || {
  echo "usage: $0 [-n] <file.md>" >&2
  exit 1
}

NOW=$(date +%Y-%m-%dT%H:%M:%S%:z)
TODAY=${NOW%%T*}

DIR=$(dirname "$FILE")
BASE=$(basename "$FILE")
SLUG=$(sed -E 's/^[0-9]{4}-[0-9]{2}-[0-9]{2}-//' <<<"$BASE")
TARGET="$DIR/$TODAY-$SLUG"

if ((DRY_RUN)); then
  echo "date  -> $NOW"
  echo "draft -> false"
  echo "mv    -> $TARGET"
  exit 0
fi

# Only the FIRST match (0,/re/) so body lines are left untouched.
sed -i -E \
  -e "0,/^date:/ s|^date:.*|date: '$NOW'|" \
  -e "0,/^draft:/ s|^draft:.*|draft: false|" \
  "$FILE"

if [[ "$FILE" != "$TARGET" ]]; then
  if git -C "$DIR" rev-parse --is-inside-work-tree &>/dev/null; then
    git mv "$FILE" "$TARGET"
  else
    mv "$FILE" "$TARGET"
  fi
fi

echo "published: $TARGET ($NOW)"
