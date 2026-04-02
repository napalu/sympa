#!/usr/bin/env bash
# Called by goreleaser signs section. Only signs Mach-O binaries.
set -euo pipefail

: "${CODESIGN_IDENTITY:=-}"
: "${CODESIGN_IDENTIFIER:=tech.heyworth.sympa}"
artifact="$1"

if file "$artifact" | grep -q "Mach-O"; then
  codesign -s "$CODESIGN_IDENTITY" \
    --identifier "$CODESIGN_IDENTIFIER" \
    --options runtime \
    --timestamp \
    "$artifact"
  echo "Signed: $artifact"
else
  echo "Skipping non-macOS binary: $artifact"
fi
