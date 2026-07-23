#!/usr/bin/env bash
#
# new.sh - create a new Hugo post draft named YYYY-MM-DD-slug.md
#
# Usage: ./new.sh montanha-ascese
#        ./new.sh -n montanha-ascese    # dry run: only print what would happen

set -euo pipefail

POSTS_DIR="content/posts"

DRY_RUN=0
[[ "${1:-}" == "-n" ]] && {
  DRY_RUN=1
  shift
}

SLUG="${1:-}"
[[ -n "$SLUG" ]] || {
  echo "usage: $0 [-n] <slug>" >&2
  exit 1
}

# Strip a leading date if the slug was pasted with one, and drop any .md suffix.
SLUG=$(sed -E -e 's/^[0-9]{4}-[0-9]{2}-[0-9]{2}-//' -e 's/\.md$//' <<<"$SLUG")

TODAY=$(date +%F)
TARGET="$POSTS_DIR/$TODAY-$SLUG.md"

[[ -e "$TARGET" ]] && {
  echo "file already exists: $TARGET" >&2
  exit 1
}

if ((DRY_RUN)); then
  echo "hugo new content -> $TARGET"
  exit 0
fi

hugo new content "$TARGET"

echo "created: $TARGET"
