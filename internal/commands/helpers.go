package commands

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/napalu/sympa/internal/age"
	"github.com/napalu/sympa/internal/agent"
	"github.com/napalu/sympa/internal/store"
)

var errNoConfig = fmt.Errorf("internal error: could not get config from parser context")

func requireInit(s *store.Store) error {
	if !s.IsInitialized() {
		return fmt.Errorf("store not initialized (run 'sympa init' first)")
	}
	return nil
}

func requireNoActiveRekey(s *store.Store) error {
	journal := filepath.Join(s.Dir(), rekeyJournalFile)
	if _, err := os.Stat(journal); err == nil {
		return fmt.Errorf("an interrupted rekey is pending\nrun 'sympa keyfile rekey --resume' to finish or '--abort' to roll back before making changes")
	}
	return nil
}

// requireKeyfileConsistency checks that the provided keyfile matches the store's expectation.
func requireKeyfileConsistency(s *store.Store, keyfilePath string) error {
	meta, err := s.ReadMetadata()
	if err != nil {
		return err
	}

	hasStoreFingerprint := meta.KeyfileFingerprint != ""
	hasKeyfile := keyfilePath != ""

	switch {
	case hasStoreFingerprint && !hasKeyfile:
		return fmt.Errorf("this store requires a keyfile (set --keyfile or SYMPA_KEYFILE)")
	case !hasStoreFingerprint && hasKeyfile:
		return fmt.Errorf("this store was initialized without a keyfile; remove --keyfile")
	case hasStoreFingerprint && hasKeyfile:
		content, err := os.ReadFile(keyfilePath)
		if err != nil {
			return fmt.Errorf("reading keyfile: %w", err)
		}
		fp := age.KeyfileFingerprint(content)
		if fp != meta.KeyfileFingerprint {
			return fmt.Errorf("keyfile mismatch:\n  store expects: %s\n  keyfile has:   %s", meta.KeyfileFingerprint, fp)
		}
	}
	return nil
}

// applyKeyfile derives a passphrase via HKDF if keyfilePath is non-empty.
func applyKeyfile(passphrase, keyfilePath string) (string, error) {
	if keyfilePath == "" {
		return passphrase, nil
	}
	content, err := os.ReadFile(keyfilePath)
	if err != nil {
		return "", fmt.Errorf("reading keyfile: %w", err)
	}
	return age.DerivePassphrase(passphrase, content)
}

// cachedDecrypt decrypts ciphertext using the agent cache with global passphrase fallback.
// Tries: (1) per-secret cached passphrase, (2) global cached passphrase, (3) terminal prompt.
// On success, caches the passphrase under both the secret path and the global ("") key.
// Returns the plaintext and the passphrase used (callers that re-encrypt need the passphrase).
func cachedDecrypt(ciphertext []byte, secretPath, prompt, keyfilePath string) ([]byte, string, error) {
	if agent.Enabled() {
		c := agent.NewClient()
		if err := c.EnsureRunning(); err == nil {
			// Try secret-specific cache
			if pass, ok := c.Get(secretPath); ok {
				if pt, err := age.Decrypt(ciphertext, pass); err == nil {
					return pt, pass, nil
				}
			}
			// Try global passphrase fallback
			if pass, ok := c.Get(""); ok {
				if pt, err := age.Decrypt(ciphertext, pass); err == nil {
					cachePassphrase(secretPath, pass)
					return pt, pass, nil
				}
			}
		}
	}

	raw, err := age.ReadPassphrase(prompt)
	if err != nil {
		return nil, "", err
	}
	pass, err := applyKeyfile(raw, keyfilePath)
	if err != nil {
		return nil, "", err
	}

	pt, err := age.Decrypt(ciphertext, pass)
	if err != nil {
		return nil, "", err
	}

	cachePassphrase(secretPath, pass)
	cachePassphrase("", pass)
	return pt, pass, nil
}

// cachedPassphrase checks the agent for a cached passphrase, falling back to terminal prompt.
// When SYMPA_AGENT=off, the agent is bypassed entirely.
// The returned passphrase already has keyfile derivation applied (if keyfilePath is set).
func cachedPassphrase(secretPath, prompt, keyfilePath string) (string, error) {
	if agent.Enabled() {
		c := agent.NewClient()
		if err := c.EnsureRunning(); err == nil {
			if pass, ok := c.Get(secretPath); ok {
				return pass, nil
			}
		}
	}
	raw, err := age.ReadPassphrase(prompt)
	if err != nil {
		return "", err
	}
	return applyKeyfile(raw, keyfilePath)
}

// confirmPassphrase prompts for passphrase confirmation with optional keyfile derivation.
func confirmPassphrase(keyfilePath string, prompt ...string) (string, error) {
	raw, err := age.ReadPassphraseConfirm(prompt...)
	if err != nil {
		return "", err
	}
	if len(raw) < 8 && keyfilePath == "" {
		fmt.Fprintln(os.Stderr, "Warning: passphrase is shorter than 8 characters and no keyfile is configured.")
		fmt.Fprintln(os.Stderr, "Short passphrases are vulnerable to brute-force attacks.")
	}
	return applyKeyfile(raw, keyfilePath)
}

// writePassphrase handles passphrase acquisition for write operations.
// When SYMPA_AGENT_MODE=rw, it checks the agent cache before prompting.
// The first write prompts with confirmation; subsequent writes reuse the cached passphrase.
func writePassphrase(secretPath, keyfilePath string, prompt ...string) (string, error) {
	// Only check cache if agent is already running — don't spawn before prompting,
	// because the spawn can interfere with /dev/tty passphrase reads on piped stdin.
	if agent.WriteEnabled() && agent.Enabled() {
		c := agent.NewClient()
		if c.Ping() {
			if pass, ok := c.Get(secretPath); ok {
				return pass, nil
			}
			if pass, ok := c.Get(""); ok {
				cachePassphrase(secretPath, pass)
				return pass, nil
			}
		}
	}
	// No cache hit or agent not running — prompt first, then cache
	pass, err := confirmPassphrase(keyfilePath, prompt...)
	if err != nil {
		return "", err
	}
	if agent.WriteEnabled() {
		cachePassphrase("", pass)
	}
	cachePassphrase(secretPath, pass)
	return pass, nil
}

// cachePassphrase stores a passphrase in the agent (best effort, silent on failure).
// When SYMPA_AGENT=off, this is a no-op.
// Auto-starts the agent if it's not already running.
func cachePassphrase(secretPath, passphrase string) {
	if !agent.Enabled() {
		return
	}
	c := agent.NewClient()
	if err := c.EnsureRunning(); err != nil {
		return
	}
	c.Set(secretPath, passphrase)
}

// verifyPassphraseConsistency picks a random existing secret and tries to decrypt it
// with the given passphrase. If decryption fails, a warning is printed to stderr.
// This is purely advisory — it never returns an error or blocks the operation.
func verifyPassphraseConsistency(s *store.Store, passphrase, excludeName string) {
	if strings.EqualFold(os.Getenv("SYMPA_NO_VERIFY"), "true") {
		return
	}

	secrets, err := s.AllSecrets()
	if err != nil || len(secrets) == 0 {
		return
	}

	// Filter out the secret being written
	var candidates []string
	for _, name := range secrets {
		if name != excludeName {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		return
	}

	// Pick a random candidate
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(candidates))))
	if err != nil {
		return
	}
	picked := candidates[idx.Int64()]

	ciphertext, err := s.Read(picked)
	if err != nil {
		return
	}

	if _, err := age.Decrypt(ciphertext, passphrase); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: passphrase does not match existing secrets.")
		fmt.Fprintln(os.Stderr, "If this is intentional, you can ignore this warning.")
	}
}
