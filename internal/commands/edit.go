package commands

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/age"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/secret"
	"github.com/napalu/sympa/internal/store"
)

func handleEdit(p *goopt.Parser, _ *goopt.Command) error {
	cfg, ok := goopt.GetStructCtxAs[*options.Config](p)
	if !ok {
		return errNoConfig
	}
	s := store.New()
	if err := requireInit(s); err != nil {
		return err
	}
	if err := requireNoActiveRekey(s); err != nil {
		return err
	}
	if err := requireKeyfileConsistency(s, cfg.Keyfile); err != nil {
		return err
	}

	tmp, err := secret.Create()
	if err != nil {
		return err
	}
	defer tmp.Cleanup()

	exists := s.Exists(cfg.Edit.Name)
	var passphrase string

	if exists {
		ciphertext, err := s.Read(cfg.Edit.Name)
		if err != nil {
			return err
		}
		var plaintext []byte
		plaintext, passphrase, err = cachedDecrypt(ciphertext, cfg.Edit.Name, "Enter passphrase: ", cfg.Keyfile)
		if err != nil {
			return err
		}
		if err := os.WriteFile(tmp.Path, plaintext, 0600); err != nil {
			return fmt.Errorf("writing temp file: %w", err)
		}
	}

	beforeHash, err := hashFile(tmp.Path)
	if err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, tmp.Path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	afterHash, err := hashFile(tmp.Path)
	if err != nil {
		return err
	}
	if beforeHash == afterHash {
		fmt.Fprintln(os.Stderr, "No changes made.")
		return nil
	}

	plaintext, err := os.ReadFile(tmp.Path)
	if err != nil {
		return fmt.Errorf("reading edited file: %w", err)
	}

	// For new secrets or if user wants a new passphrase, prompt for confirmation
	if !exists {
		passphrase, err = writePassphrase(cfg.Edit.Name, cfg.Keyfile)
		if err != nil {
			return err
		}
	}

	ciphertext, err := age.Encrypt(plaintext, passphrase)
	if err != nil {
		return err
	}

	if err := s.Write(cfg.Edit.Name, ciphertext); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Secret %q saved.\n", cfg.Edit.Name)
	return nil
}

func hashFile(path string) ([sha256.Size]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(data), nil
}
