# sympa

A pass-like password store built on [age](https://github.com/FiloSottile/age), using symmetric passphrase encryption instead of key files.

sympa keeps the workflow and on-disk layout familiar to users of [pass](https://www.passwordstore.org/), 
but replaces GPG key-based encryption with [age's](https://github.com/FiloSottile/age) scrypt passphrase mode and uses a simple structured plaintext format for secret fields.

sympa uses age's scrypt passphrase mode exclusively — no asymmetric keys, no identity files. An optional keyfile can be combined with the passphrase via HKDF-SHA256 for an entropy 
floor independent of passphrase quality. This is a deliberate hedge against harvest-now-decrypt-later attacks in anticipation of quantum computing threats to public-key cryptography.

## Install

```bash
go install github.com/napalu/sympa/cmd/sympa@latest
```

### Build from Source

```bash
git clone https://github.com/napalu/sympa.git
cd sympa
make install
```

On macOS with a Developer ID:

```bash
export CODESIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
make install
```

### macOS: Code Signing

Recent versions of macOS increasingly enforce code signing through AMFI (Apple Mobile File Integrity). An unsigned Go binary may work initially but can be killed without warning (`SIGKILL`) after a background policy update or heuristic change — even without an OS upgrade.

If you encounter `SIGKILL` when running sympa on macOS, sign the binary:

```bash
# Ad-hoc signature (no Apple developer account needed)
codesign -s - $(go env GOPATH)/bin/sympa
```

If you have an Apple Developer ID, use a proper signature for a more robust setup:

```bash
# Build, sign, then install (codesign needs write access to the directory)
go build -o /tmp/sympa ./cmd/sympa
codesign -s "Developer ID Application: Your Name (TEAMID)" --identifier com.example.sympa /tmp/sympa
sudo cp /tmp/sympa /usr/local/bin/sympa
```

**If sympa was previously blocked**, macOS caches the rejection by path. Clear it before re-signing:

```bash
sudo spctl --remove /usr/local/bin/sympa
```

A `make install` target is provided that handles the build-sign-install sequence. On macOS, set `CODESIGN_IDENTITY` to use a Developer ID signature; without it, ad-hoc signing (`-s -`) is used automatically. On Linux, code signing is skipped.

## Quick Start

```bash
# Initialize the store
sympa init

# Add a secret
sympa insert email/gmail

# Show it
sympa show email/gmail

# Copy password to clipboard (auto-clears after 45s)
sympa show -c email/gmail

# Edit with $EDITOR
sympa edit email/gmail

# Generate a random password
sympa generate ssh/server -l 64

# List everything
sympa ls
```

## Commands

| Command | Description |
|---------|-------------|
| `sympa init` | Initialize a new store at `~/.sympa` |
| `sympa ls [subfolder]` | List secrets as a tree |
| `sympa show [-c] [-f field] <name>` | Decrypt and display a secret |
| `sympa insert [-m] [--no-verify] <name>` | Insert a new secret (`-m` for multiline) |
| `sympa edit <name>` | Edit a secret with `$EDITOR` |
| `sympa rm [-rf] <name>` | Remove a secret or directory |
| `sympa mv <src> <dst>` | Move or rename a secret |
| `sympa cp <src> <dst>` | Copy a secret (re-encrypts with new passphrase) |
| `sympa find <pattern>` | Search secret names (case-insensitive) |
| `sympa grep <pattern>` | Search within decrypted contents |
| `sympa generate [-lnc] [--no-verify] <name>` | Generate and store a random password |
| `sympa totp [-c] <name>` | Generate a TOTP code from a stored secret |
| `sympa git <args...>` | Run git commands on the store |
| `sympa agent start` | Start the passphrase caching agent |
| `sympa agent stop` | Stop the agent and clear cached passphrases |
| `sympa agent status` | Show whether the agent is running |
| `sympa keyfile verify` | Check keyfile matches store fingerprint |
| `sympa keyfile generate <path>` | Generate a new random keyfile (`--bytes` for size) |
| `sympa keyfile rekey [new-keyfile]` | Re-encrypt all secrets with new keyfile or passphrase |
| `sympa completion <shell>` | Generate shell completion script (bash, zsh, fish) |

Use `sympa <command> --help` for detailed usage.

## Secret Format

Secrets are plaintext before encryption. The format is simple:

```
mysecretpassword
user: florent
totp: JBSWY3DPEHPK3PXP
url: https://gmail.com
recovery: ABCD-1234
recovery: EFGH-5678
```

- **Line 1** is always the password
- Remaining lines are optional `key: value` fields — name them whatever you want
- `totp` is the only field with special meaning (used by `sympa totp` to compute codes)
- Multiple lines with the same key are fine (e.g. recovery codes)
- No fields at all? That's fine — everything still works

### Accessing Fields

```bash
sympa show email/gmail              # print everything
sympa show -c email/gmail           # copy password to clipboard
sympa show -f user email/gmail      # print just the username
sympa show -c -f user email/gmail   # copy username to clipboard
```

## TOTP

Store a TOTP key as a field in any secret:

```
mypassword
user: florent
totp: JBSWY3DPEHPK3PXP
```

Then generate codes:

```bash
sympa totp email/gmail              # print 6-digit code
sympa totp -c email/gmail           # copy code to clipboard
```

Uses RFC 6238 defaults: HMAC-SHA1, 6 digits, 30-second period. When a service shows you a QR code for TOTP setup, click "can't scan?" to get the text key, and paste it as the `totp:` field value.

## Clipboard

The `-c` flag copies to the clipboard instead of printing to stdout. The clipboard is automatically cleared after 45 seconds — but only if its contents haven't been changed since the copy.

Works on:
- **macOS** — `pbcopy`/`pbpaste`
- **Linux** — `wl-copy`/`wl-paste` (Wayland) or `xclip`/`xsel` (X11)

## Passphrase Caching Agent

sympa includes an optional caching agent (similar to `ssh-agent` or `gpg-agent`) that remembers passphrases for a configurable duration. This lets you run multiple commands without re-entering the passphrase each time.

The agent starts automatically on first use — no setup needed:

```bash
sympa show email/gmail    # prompts for passphrase, caches it
sympa show email/gmail    # no prompt — cache hit
sympa totp email/gmail    # no prompt — same secret, still cached
```

Passphrases are cached per-secret, so different secrets can use different passphrases.

### Manual Control

```bash
sympa agent status        # check if the agent is running
sympa agent stop          # stop the agent and clear all cached passphrases
sympa agent start         # start manually (usually not needed)
```

### Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SYMPA_AGENT_TIMEOUT` | Cache TTL as a Go duration (e.g. `5m`, `30s`, `1h`) | `2m` |
| `SYMPA_AGENT_SOCK` | Custom Unix socket path | Auto-detected |
| `SYMPA_AGENT=off` | Disable the agent entirely (paranoid mode) | Enabled |
| `SYMPA_AGENT_MODE` | Agent cache mode: `r` (read-only) or `rw` (read-write). Also available as `sympa agent --mode` | `r` |

Socket path resolution: `$SYMPA_AGENT_SOCK` > `$XDG_RUNTIME_DIR/sympa/agent.sock` > `/tmp/sympa-<uid>/agent.sock`.

### Paranoid Mode

If you prefer to never cache passphrases and always prompt:

```bash
export SYMPA_AGENT=off
```

With this set, sympa behaves exactly as it does without the agent — every operation prompts for a passphrase via the terminal. See [SECURITY.md](SECURITY.md) for the full threat model.

### Write Mode (Bulk Import)

By default, the agent only caches passphrases for read operations. Write operations (insert, generate, edit, cp) always prompt with confirmation. For bulk import scenarios, enable read-write mode:

```bash
export SYMPA_AGENT_MODE=rw
```

In `rw` mode, the first write operation prompts with confirmation as usual. Subsequent writes reuse the cached passphrase automatically. This is ideal for migrating from another password store:

```bash
export SYMPA_AGENT_MODE=rw
export SYMPA_KEYFILE=/path/to/keyfile    # optional

# Migrate all secrets from pass to sympa
STORE="${PASSWORD_STORE_DIR:-$HOME/.password-store}"
while IFS= read -r gpg; do
  secret="${gpg#$STORE/}"
  secret="${secret%.gpg}"
  pass show "$secret" | sympa insert -f "$secret"
done < <(find "$STORE" -name "*.gpg" -type f)

unset SYMPA_AGENT_MODE
```

The first `sympa insert` prompts for a passphrase (with confirmation). All subsequent inserts reuse it from the agent cache. Piped input receives EOF automatically when the source command finishes — no Ctrl+D needed. Remember to unset `SYMPA_AGENT_MODE` when done.

## Passphrase Verification

When inserting or generating a secret, sympa picks a random existing secret and tries to decrypt it with the passphrase you just entered. If decryption fails, a warning is printed:

```
Warning: passphrase does not match existing secrets.
If this is intentional, you can ignore this warning.
```

This catches the case where you accidentally use a different passphrase than the rest of your store. The warning is advisory — it never blocks the operation, since you may legitimately want a different passphrase for a specific secret.

Verification is skipped automatically when the store is empty or the only secret is the one being written.

To disable verification:

```bash
# Per-command flag
sympa insert --no-verify email/alternate
sympa generate --no-verify ssh/special

# Environment variable (applies globally)
export SYMPA_NO_VERIFY=true
```

## Enhanced Security: Keyfile

sympa supports an optional keyfile that is combined with your passphrase using HKDF-SHA256 to derive the actual encryption key. This provides an entropy floor independent of passphrase quality — even a weak passphrase becomes resistant to brute-force when combined with a 256-bit random keyfile.

### Setup

```bash
# Initialize with a keyfile (generates one if absent)
sympa --keyfile /path/to/keyfile init

# Or generate a keyfile first (default 32 bytes, use --bytes for more)
sympa keyfile generate /path/to/keyfile
sympa keyfile generate --bytes 64 /path/to/keyfile
sympa --keyfile /path/to/keyfile init
```

### Ongoing Usage

Set the environment variable to avoid passing `--keyfile` on every command:

```bash
export SYMPA_KEYFILE=/path/to/keyfile

sympa insert email/gmail     # keyfile applied automatically
sympa show email/gmail       # keyfile applied automatically
```

The keyfile is only needed at passphrase prompt time. The agent caches the derived passphrase, so subsequent operations use the cache as normal.

### Keyfile Verification

```bash
SYMPA_KEYFILE=/path/to/keyfile sympa keyfile verify
```

### Rekeying

Re-encrypt all secrets when changing your keyfile or passphrase:

```bash
# Swap to a stronger keyfile
sympa keyfile generate --bytes 64 /path/to/new-keyfile
sympa keyfile rekey -k /path/to/old-keyfile /path/to/new-keyfile

# Add a keyfile to a store that didn't have one
sympa keyfile rekey /path/to/keyfile

# Remove a keyfile
sympa keyfile rekey -k /path/to/keyfile --remove

# Change passphrase (with or without keyfile)
sympa keyfile rekey --passphrase
sympa keyfile rekey -k /path/to/keyfile --passphrase

# Combined: swap keyfile and change passphrase
sympa keyfile rekey -k /path/to/old-keyfile /path/to/new-keyfile --passphrase
```

After swapping or adding a keyfile, update your environment:

```bash
export SYMPA_KEYFILE=/path/to/new-keyfile
sympa keyfile verify                      # confirm the new keyfile matches
```

Rekey is **interruption-safe**. All secrets are backed up before any modifications. If the process is interrupted (Ctrl+C, power loss), use `--resume` to finish or `--abort` to roll back:

```bash
sympa keyfile rekey --resume /path/to/new-keyfile   # continue where it left off
sympa keyfile rekey --abort                         # restore from backup
```

If your store contains secrets with different passphrases, rekey detects this automatically and prompts for each distinct passphrase. Use `--passphrase` to normalize all secrets to a single new passphrase.

### Important

- **Loss of keyfile = loss of all secrets.** Back it up to separate media.
- The keyfile is stored with mode `0400` (owner read-only).
- The store's marker file records the keyfile fingerprint (SHA-256), so sympa gives clear errors if you forget the keyfile or use the wrong one.
- Stores initialized without a keyfile continue to work as before — the feature is fully opt-in.

## Store Layout

```
~/.sympa/
├── .sympa-store          # Marker file (JSON metadata when using keyfile)
├── email/
│   ├── gmail.age
│   └── work.age
├── ssh/
│   └── server.age
└── .git/                 # Optional
```

Secrets are stored as individual `.age` files encrypted with scrypt. The store location can be overridden with `$SYMPA_DIR`.


## Design Decisions

### Symmetric-only encryption

Unlike pass, which encrypts secrets to GPG keys, sympa uses age in passphrase (scrypt) mode exclusively.

Each secret is protected by a passphrase chosen at encryption time, and no private keys are stored on disk. This removes the need for keyrings, identity files, 
or trust configuration, and keeps each secret independent.

Security depends primarily on the entropy of the passphrase (and keyfile, if configured) and the cost of offline guessing attacks. Symmetric cryptography is generally considered less affected by future quantum attacks than classical public-key systems such as RSA and ECC. Grover's algorithm provides a quadratic speedup for brute-force search, effectively halving the security level — a keyfile mitigates this by providing a high entropy floor independent of passphrase quality.

age also supports public-key and hybrid post-quantum recipients, but these modes introduce additional key management and larger metadata. For the current scope of sympa, 
a symmetric design is considered the most predictable and conservative choice.

### No automatic git integration

`sympa git` is a thin passthrough to `git -C <store-dir>`.

sympa does not automatically commit, push, or manage history. Version control is left entirely to the user to avoid hidden behavior that could expose secrets or metadata unexpectedly.

### Temporary files in RAM when possible

`sympa edit` decrypts secrets to RAM-backed storage when available.

- On Linux, `/dev/shm` is used if present.
- On macOS, a temporary RAM disk is created via `hdiutil`.
- If no RAM-backed storage is available, the system temporary directory is used as a fallback.

Temporary files are overwritten with random data before deletion as a best-effort mitigation against recovery from disk. This does
not guarantee secure erasure on all filesystems, but reduces the risk of leaving plaintext on persistent storage.

### One secret per file

Like pass, each secret is stored as a separate file in a directory tree.

Files are encrypted individually and stored with the `.age` extension.  
Before encryption, the plaintext uses a simple line-based format:

- line 1 is the password
- additional lines may contain `key: value` fields
- no fixed schema is required

This keeps secrets easy to inspect, diff, and version-control.

### Optional passphrase caching agent

sympa includes an optional caching agent that stores passphrases in memory for a short time.

The agent runs as a local Unix socket service owned by the current user. Any process running as the same user may be able
to access the agent, so this feature trades some isolation for convenience. It can be disabled entirely by setting `SYMPA_AGENT=off`.

When disabled, sympa prompts for a passphrase for every operation.

### Scope

sympa is intentionally close to pass in workflow and storage layout, but differs in a few key areas:

- encryption uses age passphrase mode instead of GPG keys
- no private keys or identity files are stored on disk
- each secret is encrypted independently with a passphrase
- plaintext format allows simple `key: value` fields

The goal is to keep the pass mental model while reducing key management complexity and avoiding reliance on long-term public-key security assumptions.

sympa is not intended to be a full secret manager, vault, or keychain replacement.  
It is a small CLI tool for storing encrypted information locally with minimal abstraction.

## Security Considerations

sympa is a small tool with a simple threat model. Understanding its boundaries helps you use it well.

### Passphrase strength matters — unless you use a keyfile

Without a keyfile, security depends entirely on your passphrase entropy and age's scrypt cost (work factor 2^18). A high-entropy passphrase makes brute-force search impractical with current technology; a short or predictable one does not. sympa warns when a passphrase is shorter than 8 characters and no keyfile is configured.

With a keyfile, the passphrase and keyfile content are combined via HKDF-SHA256 to derive the encryption key. The keyfile provides an entropy floor independent of passphrase quality — even a weak passphrase becomes resistant to brute-force when combined with a random keyfile. A 32-byte keyfile (the default) provides 256 bits of entropy, which remains strong even accounting for Grover's algorithm quadratic speedup in a post-quantum context (~128-bit effective security). A 64-byte keyfile raises this to ~256-bit effective post-quantum security. If you want defense in depth without relying solely on passphrase quality, use a keyfile.

Write operations require typing the passphrase twice (type + confirm), which mitigates accidental typos. Additionally, `insert` and `generate` verify the passphrase against a random existing secret and warn if it doesn't match (see [Passphrase Verification](#passphrase-verification)). However, if you consistently mistype and confirm the same wrong passphrase, that secret is encrypted with a passphrase you may not remember. There is no recovery mechanism — treat your passphrase (and keyfile, if used) as irreplaceable.

### Metadata is not encrypted

File names, directory structure, and modification times are visible in plaintext on disk. An attacker with read access to `~/.sympa` can see that `banking/schwab.age` and `ssh/prod-server.age` exist without decrypting anything. This is inherent to the pass-style one-file-per-secret model. If secret names are sensitive, use opaque names (e.g., `a1.age`) or keep the store on an encrypted volume.

### TOTP seeds and passwords share a passphrase

Storing a `totp:` field alongside the password means a single passphrase compromise reveals both factors, effectively reducing two-factor authentication to one factor. This is a convenience trade-off that many password managers make (1Password, Bitwarden, etc.). If you require strict factor separation, store TOTP seeds in a dedicated hardware token or separate app.

### Memory and swap exposure

Go does not zero strings after use and does not support `mlock` on all platforms. Passphrases and decrypted secrets live in heap memory until garbage collection. On systems without encrypted swap (common on Linux by default), this data could be paged to disk. Mitigations:

- **macOS**: FileVault encrypts swap by default. No action needed.
- **Linux**: Enable encrypted swap, or use `zram` swap (RAM-only). Alternatively, set `SYMPA_AGENT=off` and minimize the window during which secrets are in memory.
- **All platforms**: `sympa edit` uses RAM-backed temporary files (`/dev/shm` on Linux, RAM disk on macOS) to avoid writing plaintext to persistent storage. Temp files are overwritten with random data before deletion.

### Agent trust boundary

The caching agent runs as a Unix socket service owned by the current user (permissions `0600`). Any process running as the same user can query the socket and retrieve cached passphrases. This is the same trust model as `ssh-agent` and `gpg-agent`. If this is unacceptable, disable the agent entirely:

```bash
export SYMPA_AGENT=off
```

### Clipboard sharing

The `-c` flag copies secrets to the system clipboard, which is accessible to all processes running as the current user. The clipboard is auto-cleared after 45 seconds, but only if its contents haven't changed since the copy. Avoid `-c` on shared or untrusted systems.

### No built-in sync or backup

sympa does not automatically commit, push, or replicate your store. A disk failure without backup means total loss. Use `sympa git` to version-control the store and push to a private remote:

```bash
sympa git init
sympa git remote add origin git@github.com:you/secrets.git
```

Encrypted `.age` files are safe to store on remote servers — they reveal nothing without the passphrase (and keyfile, if configured). Directory structure and file names are visible, however (see "Metadata is not encrypted" above).

## Shell Completion

sympa generates completion scripts for bash, zsh, and fish. Completions include commands, flags, and dynamic secret path completion (tab-completing secret names from your store).

### Install

```bash
# Bash
sympa completion bash > ~/.local/share/bash-completion/completions/sympa

# Zsh (add completion directory to fpath if not already)
sympa completion zsh > "${fpath[1]}/_sympa"

# Fish
sympa completion fish > ~/.config/fish/completions/sympa.fish
```

Restart your shell or source the script to activate. Secret names complete dynamically — as you add or remove secrets, tab completion reflects the current store contents.

## Git Integration

```bash
sympa git init
sympa git remote add origin git@github.com:you/secrets.git
sympa git add -A
sympa git commit -m "initial"
sympa git push -u origin main
```

## Dependencies

- [filippo.io/age](https://github.com/FiloSottile/age) — encryption
- [github.com/napalu/goopt/v2](https://github.com/napalu/goopt/tree/main/v2) — CLI parsing
- [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) — secure passphrase input

## License

MIT
