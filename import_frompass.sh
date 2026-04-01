#!/bin/bash

export SYMPA_AGENT_MODE="rw"
export SYMPA_KEYFILE="/tmp/test.key"    # optional

# Migrate all secrets from pass to sympa
STORE="${PASSWORD_STORE_DIR:-$HOME/.password-store}"
while IFS= read -r gpg; do
  secret="${gpg#$STORE/}"
  secret="${secret%.gpg}"
  pass show "$secret" | ./sympa insert -f "$secret"
done < <(find "$STORE" -name "*.gpg" -type f)

unset SYMPA_AGENT_MODE
