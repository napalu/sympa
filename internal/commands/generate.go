package commands

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/age"
	"github.com/napalu/sympa/internal/clip"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

const (
	alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	symbols      = "!@#$%^&*()-_=+[]{}|;:,.<>?/"
)

func handleGenerate(p *goopt.Parser, _ *goopt.Command) error {
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

	if s.Exists(cfg.Generate.Name) && !cfg.Generate.Force {
		return fmt.Errorf("secret %q already exists (use -f to overwrite)", cfg.Generate.Name)
	}

	if cfg.Generate.Length <= 0 {
		return fmt.Errorf("password length must be positive")
	}

	charset := alphanumeric
	if !cfg.Generate.NoSymbols {
		charset += symbols
	}

	password, err := generatePassword(cfg.Generate.Length, charset)
	if err != nil {
		return fmt.Errorf("generating password: %w", err)
	}

	passphrase, err := writePassphrase(cfg.Generate.Name, cfg.Keyfile)
	if err != nil {
		return err
	}

	if !cfg.Generate.NoVerify {
		verifyPassphraseConsistency(s, passphrase, cfg.Generate.Name)
	}

	ciphertext, err := age.Encrypt([]byte(password), passphrase)
	if err != nil {
		return err
	}

	if err := s.Write(cfg.Generate.Name, ciphertext); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Generated password for", cfg.Generate.Name)

	if cfg.Generate.Clip {
		return clip.CopyAndClear(password, clip.ClearTimeout)
	}

	fmt.Println(password)
	return nil
}

func generatePassword(length int, charset string) (string, error) {
	result := make([]byte, length)
	max := big.NewInt(int64(len(charset)))
	for i := range result {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}
