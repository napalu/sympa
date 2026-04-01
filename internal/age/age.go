package age

import (
	"bytes"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"filippo.io/age"
	"golang.org/x/term"
)

// DerivePassphrase combines a passphrase with keyfile content using HKDF-SHA256,
// returning a hex-encoded 32-byte derived key suitable for use as an age passphrase.
func DerivePassphrase(passphrase string, keyfileContent []byte) (string, error) {
	derived, err := hkdf.Key(sha256.New, []byte(passphrase), keyfileContent, "sympa-v1", 32)
	if err != nil {
		return "", fmt.Errorf("deriving passphrase: %w", err)
	}
	return hex.EncodeToString(derived), nil
}

// KeyfileFingerprint returns the SHA-256 fingerprint of keyfile content.
func KeyfileFingerprint(content []byte) string {
	h := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(h[:])
}

// Encrypt encrypts plaintext using a scrypt passphrase.
func Encrypt(plaintext []byte, passphrase string) ([]byte, error) {
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return nil, fmt.Errorf("creating scrypt recipient: %w", err)
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return nil, fmt.Errorf("initializing encryption: %w", err)
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, fmt.Errorf("writing encrypted data: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("finalizing encryption: %w", err)
	}
	return buf.Bytes(), nil
}

// Decrypt decrypts ciphertext using a scrypt passphrase.
func Decrypt(ciphertext []byte, passphrase string) ([]byte, error) {
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, fmt.Errorf("creating scrypt identity: %w", err)
	}

	r, err := age.Decrypt(bytes.NewReader(ciphertext), identity)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading decrypted data: %w", err)
	}
	return plaintext, nil
}

// ReadPassphrase prompts the user and reads a passphrase with echo disabled.
// When stdin is not a terminal (e.g. piped input), the passphrase is read
// from /dev/tty instead, matching the behavior of GPG and SSH.
func ReadPassphrase(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)

	fd := int(os.Stdin.Fd())
	var tty *os.File
	if !term.IsTerminal(fd) {
		var err error
		tty, err = os.Open("/dev/tty")
		if err != nil {
			return "", fmt.Errorf("cannot open terminal for passphrase: %w", err)
		}
		defer tty.Close()
		fd = int(tty.Fd())
	}

	passphrase, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading passphrase: %w", err)
	}
	return string(passphrase), nil
}

// ReadPassphraseConfirm prompts for a passphrase twice and verifies they match.
// An optional prompt can be provided for the first prompt; defaults to "Enter passphrase: ".
func ReadPassphraseConfirm(prompt ...string) (string, error) {
	p := "Enter passphrase: "
	if len(prompt) > 0 && prompt[0] != "" {
		p = prompt[0]
	}
	p1, err := ReadPassphrase(p)
	if err != nil {
		return "", err
	}
	if p1 == "" {
		return "", fmt.Errorf("passphrase cannot be empty")
	}
	p2, err := ReadPassphrase("Confirm passphrase: ")
	if err != nil {
		return "", err
	}
	if p1 != p2 {
		return "", fmt.Errorf("passphrases do not match")
	}
	return p1, nil
}
