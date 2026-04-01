# Security Model

## Encryption

sympa uses [age](https://github.com/FiloSottile/age) in scrypt passphrase mode exclusively. Each secret is encrypted individually as a standalone `.age` file using a passphrase-derived key. There are no stored keys, no key files, and no asymmetric cryptography.

scrypt is a memory-hard key derivation function designed to resist brute-force attacks. Symmetric encryption (used by age's scrypt mode) is not vulnerable to the quantum computing threats that affect RSA and ECC.

### Symmetric-only design

sympa uses age in passphrase (scrypt) mode exclusively. Each secret is encrypted with a user-supplied passphrase, and no private keys are stored on disk.

This design keeps the trust model minimal: there are no identity files, no long-lived private keys, and no dependency on public-key infrastructure. Security depends
primarily on the entropy of the passphrase and the cost of offline guessing attacks.

Symmetric cryptography is generally considered more robust against quantum attacks than classical public-key systems such as RSA and ECC, but it is not immune. 
Under widely accepted models, quantum search algorithms (for example Grover’s algorithm) could reduce brute-force resistance compared to the classical setting. 
Therefore, weak or guessable passphrases remain insecure even when using symmetric encryption.

sympa intentionally does not enforce passphrase strength. Users are responsible for choosing high-entropy passphrases appropriate for their threat model.

### Post-quantum recipients

Recent versions of age support hybrid post-quantum recipients. sympa currently does not use these modes.

This is a deliberate choice. While post-quantum recipient modes provide strong long-term security properties, they also introduce tradeoffs including larger metadata, 
additional key management, and reliance on newer cryptographic constructions that have seen less real-world deployment than symmetric primitives and memory-hard KDFs.

Future versions of sympa may optionally support post-quantum recipients, but the default design is expected to remain symmetric-only unless there is a clear security or usability benefit.

## Keyfile Derivation

sympa optionally supports combining a passphrase with a keyfile using HKDF-SHA256 to derive the actual encryption passphrase fed to age's scrypt mode.

### Construction

```
derived = HKDF-SHA256(secret=passphrase, salt=keyfile, info="sympa-v1", len=32)
age_passphrase = hex(derived)
```

The keyfile provides 256 bits of entropy as the HKDF salt. The passphrase is the HKDF secret. The fixed info string `"sympa-v1"` acts as a domain separator. The 32-byte output is hex-encoded and used as the age passphrase, which then goes through age's own scrypt KDF.

### What keyfile derivation protects against

- **Weak passphrases**: even `password123` combined with a 256-bit random keyfile produces an unguessable derived key. The keyfile provides an entropy floor independent of passphrase quality.
- **Brute-force attacks**: an attacker with access to the encrypted files but not the keyfile cannot mount an offline dictionary attack against the passphrase alone.
- **Passphrase reuse**: the same passphrase used with different keyfiles produces completely different derived keys.

### What keyfile derivation does NOT protect against

- **Keyfile compromise**: if an attacker obtains both the encrypted files and the keyfile, they can brute-force the passphrase as if no keyfile were used. The keyfile is not a substitute for a strong passphrase — it is a complement.
- **Root access or memory forensics**: the derived passphrase exists in memory during encryption/decryption, subject to the same limitations as regular passphrases (see agent threat model below).
- **Loss of keyfile**: if the keyfile is lost or corrupted, all secrets encrypted with it are irrecoverable. There is no recovery mechanism.

### Storage recommendations

- Store the keyfile on separate media (USB drive, hardware token, encrypted volume)
- Set permissions to `0400` (owner read-only) — sympa does this automatically when generating
- Back up the keyfile independently from the encrypted store
- The store's `.sympa-store` marker file records a SHA-256 fingerprint of the keyfile for consistency checking, but does not contain the keyfile itself

### Fingerprint verification

The store marker file records `sha256:<hex>` of the keyfile content. On every operation, sympa verifies:
- If the store has a fingerprint but no `--keyfile` is provided → error
- If the store has no fingerprint but `--keyfile` is provided → error
- If fingerprints don't match → error with both fingerprints shown

This prevents accidental use of the wrong keyfile or forgetting to provide one.

## Passphrase Handling

- Passphrases are read from the terminal using `golang.org/x/term` with echo disabled
- When creating or re-encrypting secrets, passphrases must be confirmed (entered twice)
- Plaintext secrets are never written to disk — `sympa edit` uses RAM-backed temp files (`/dev/shm` on Linux, `hdiutil` RAM disk on macOS)
- Temp files are overwritten with random data before deletion (best-effort; this does not guarantee secure erasure on all filesystems)

## Caching Agent Threat Model

The passphrase caching agent (`sympa agent`) holds decrypted passphrases in process memory for a configurable duration (default 2 minutes). This is an explicit convenience-for-security tradeoff, similar to `ssh-agent` and `gpg-agent`.

### What the agent protects against

- **Casual shoulder-surfing**: passphrases are never displayed
- **Disk forensics**: passphrases only exist in process memory, never on disk
- **Core dump leakage**: core dumps are disabled via `RLIMIT_CORE=0`
- **Process tracing (Linux)**: `PR_SET_DUMPABLE=0` prevents `/proc/pid/mem` reads by non-root
- **Swap file leakage (Linux)**: `mlockall` is called to prevent memory from being paged to swap (best-effort, requires `CAP_IPC_LOCK` or sufficient `RLIMIT_MEMLOCK`)
- **Socket hijacking**: the socket file is created with mode `0600` in a directory with mode `0700`, and the client verifies socket ownership before connecting

### What the agent does NOT protect against

- **Root access**: a root user can read any process memory, attach debuggers, and bypass all protections. This is true of all user-space secret caching (ssh-agent, gpg-agent, etc.)
- **Same-user attacks**: another process running as your user can connect to the agent socket and retrieve cached passphrases. The agent relies on Unix file permissions for access control
- **Memory forensics**: a sufficiently privileged attacker with access to a memory dump or live system can extract cached passphrases. The agent zeroes passphrases on eviction/shutdown but cannot prevent all copies (Go runtime, OS buffers)
- **Agent impersonation**: if an attacker removes the real socket and places their own before sympa connects, they could intercept passphrases. Socket ownership verification mitigates but does not eliminate this risk in a compromised environment

### Hardening measures

| Measure | Linux | macOS | Other |
|---------|-------|-------|-------|
| Core dumps disabled (`RLIMIT_CORE=0`) | Yes | Yes | No |
| Memory locking (`mlockall`) | Yes (best-effort) | No (unavailable) | No |
| Non-dumpable process (`PR_SET_DUMPABLE=0`) | Yes | No (unavailable) | No |
| Socket permissions (`0600` in `0700` dir) | Yes | Yes | Yes |
| Socket ownership verification | Yes | Yes | No |
| Memory zeroing on eviction | Yes | Yes | Yes |

### Write mode (`rw`)

By default (`SYMPA_AGENT_MODE=r`), the agent only caches passphrases obtained during read operations. Write operations (insert, generate, edit, cp) always prompt with confirmation, even if a passphrase is cached for that secret.

Setting `SYMPA_AGENT_MODE=rw` allows write operations to reuse cached passphrases, skipping the confirmation prompt. This is designed for bulk import scenarios where the same passphrase is used for many secrets.

**Tradeoff**: in `rw` mode, a typo in the first passphrase confirmation propagates silently to all subsequent writes. This is acceptable for one-time migration workflows but should not be left enabled for daily use. The mode only takes effect when explicitly set — the default (`r`) always requires confirmation for writes.

### Disabling the agent

For environments where caching passphrases in memory is unacceptable:

```bash
export SYMPA_AGENT=off
```

This completely disables the agent — no auto-spawn, no caching. Every operation prompts for a passphrase via the terminal, exactly as if the agent feature did not exist.

### Agent protocol

The agent uses a text-based protocol over a Unix domain socket. Each connection handles one request-response pair:

| Request | Response | Description |
|---------|----------|-------------|
| `PING` | `PONG` | Health check |
| `GET <path>` | base64 or `MISS` | Retrieve cached passphrase |
| `SET <path> <base64>` | `OK` | Store passphrase |
| `FORGET <path>` | `OK` | Remove one cached passphrase |
| `CLEAR` | `OK` | Remove all cached passphrases |
| `SHUTDOWN` | `BYE` | Stop the agent |

Paths are URL-encoded, passphrases are base64-encoded. Connection timeout is 5 seconds.

## Reporting Vulnerabilities

If you find a security issue, please open an issue at https://github.com/napalu/sympa/issues.
