#!/usr/bin/env bash
set -euo pipefail

: "${CODESIGN_IDENTITY:?CODESIGN_IDENTITY is required}"
: "${CODESIGN_IDENTIFIER:=tech.heyworth.sympa}"

artifact="$1"

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
