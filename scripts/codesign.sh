#!/usr/bin/env bash
set -euo pipefail

artifact="${1:-}"
if [[ -z "$artifact" || "$artifact" == "noop" ]]; then
  exit 0
fi

: "${CODESIGN_IDENTITY:?CODESIGN_IDENTITY is required}"
: "${CODESIGN_IDENTIFIER:=tech.heyworth.sympa}"

if file "$artifact" | grep -q "Mach-O"; then
  codesign --force \
    --sign "$CODESIGN_IDENTITY" \
    --identifier "$CODESIGN_IDENTIFIER" \
    --options runtime \
    --timestamp \
    "$artifact"
  echo "Signed: $artifact"
else
  echo "Skipping non-macOS binary: $artifact"
fi
