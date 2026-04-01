package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/age"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

func handleGrep(p *goopt.Parser, _ *goopt.Command) error {
	cfg, ok := goopt.GetStructCtxAs[*options.Config](p)
	if !ok {
		return errNoConfig
	}
	s := store.New()
	if err := requireInit(s); err != nil {
		return err
	}
	if err := requireKeyfileConsistency(s, cfg.Keyfile); err != nil {
		return err
	}

	passphrase, err := cachedPassphrase("", "Enter passphrase: ", cfg.Keyfile)
	if err != nil {
		return err
	}

	names, err := s.AllSecrets()
	if err != nil {
		return err
	}

	found := false
	skipped := 0
	for _, name := range names {
		ciphertext, err := s.Read(name)
		if err != nil {
			skipped++
			continue
		}
		plaintext, err := age.Decrypt(ciphertext, passphrase)
		if err != nil {
			skipped++
			continue
		}
		cachePassphrase(name, passphrase)
		lines := strings.Split(string(plaintext), "\n")
		for i, line := range lines {
			if strings.Contains(line, cfg.Grep.Pattern) {
				fmt.Printf("%s:%d:%s\n", name, i+1, line)
				found = true
			}
		}
	}

	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "%d secret(s) skipped (wrong passphrase or read error)\n", skipped)
	}

	if !found {
		return fmt.Errorf("no matches for %q", cfg.Grep.Pattern)
	}
	return nil
}
