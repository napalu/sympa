package commands

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/age"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

func handleInit(p *goopt.Parser, _ *goopt.Command) error {
	cfg, ok := goopt.GetStructCtxAs[*options.Config](p)
	if !ok {
		return errNoConfig
	}

	s := store.New()
	meta := store.StoreMetadata{}

	if cfg.Keyfile != "" {
		// Generate keyfile if it doesn't exist
		if _, err := os.Stat(cfg.Keyfile); os.IsNotExist(err) {
			if err := generateKeyfile(cfg.Keyfile, 32); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Generated keyfile at %s\n", cfg.Keyfile)
		}

		content, err := os.ReadFile(cfg.Keyfile)
		if err != nil {
			return fmt.Errorf("reading keyfile: %w", err)
		}
		meta.KeyfileFingerprint = age.KeyfileFingerprint(content)
	}

	if err := s.InitWithMetadata(meta); err != nil {
		return err
	}

	fmt.Printf("Initialized empty sympa store at %s\n", s.Dir())
	if meta.KeyfileFingerprint != "" {
		fmt.Fprintf(os.Stderr, "Keyfile fingerprint: %s\n", meta.KeyfileFingerprint)
		fmt.Fprintln(os.Stderr, "WARNING: Loss of keyfile = loss of all secrets. Back it up!")
	}
	return nil
}

// generateKeyfile writes cryptographically random bytes to the given path.
func generateKeyfile(path string, size int) error {
	if size < 32 {
		return fmt.Errorf("keyfile size must be at least 32 bytes (got %d)", size)
	}
	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("generating keyfile: %w", err)
	}
	if err := os.WriteFile(path, key, 0400); err != nil {
		return fmt.Errorf("writing keyfile: %w", err)
	}
	return nil
}
