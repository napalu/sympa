package commands

import (
	"fmt"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/age"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

func handleCp(p *goopt.Parser, _ *goopt.Command) error {
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

	if !s.Exists(cfg.Cp.Source) {
		return fmt.Errorf("secret %q does not exist", cfg.Cp.Source)
	}
	if s.Exists(cfg.Cp.Dest) && !cfg.Cp.Force {
		return fmt.Errorf("secret %q already exists (use -f to overwrite)", cfg.Cp.Dest)
	}

	// Decrypt source
	ciphertext, err := s.Read(cfg.Cp.Source)
	if err != nil {
		return err
	}
	plaintext, _, err := cachedDecrypt(ciphertext, cfg.Cp.Source, "Passphrase for source secret: ", cfg.Keyfile)
	if err != nil {
		return err
	}

	// Re-encrypt for destination (can use same or different passphrase)
	dstPass, err := writePassphrase(cfg.Cp.Dest, cfg.Keyfile, "Passphrase for destination secret: ")
	if err != nil {
		return err
	}

	newCiphertext, err := age.Encrypt(plaintext, dstPass)
	if err != nil {
		return err
	}

	if err := s.Write(cfg.Cp.Dest, newCiphertext); err != nil {
		return err
	}
	return nil
}
