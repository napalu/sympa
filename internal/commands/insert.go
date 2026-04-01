package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/age"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
	"golang.org/x/term"
)

func handleInsert(p *goopt.Parser, _ *goopt.Command) error {
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

	if s.Exists(cfg.Insert.Name) && !cfg.Insert.Force {
		return fmt.Errorf("secret %q already exists (use -f to overwrite)", cfg.Insert.Name)
	}

	passphrase, err := writePassphrase(cfg.Insert.Name, cfg.Keyfile)
	if err != nil {
		return err
	}

	if !cfg.Insert.NoVerify {
		verifyPassphraseConsistency(s, passphrase, cfg.Insert.Name)
	}

	var secret []byte
	if cfg.Insert.Multiline {
		fmt.Fprintln(os.Stderr, "Enter secret (Ctrl+D to finish):")
		secret, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
	} else if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Piped input
		secret, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		secret = []byte(strings.TrimRight(string(secret), "\n"))
	} else {
		fmt.Fprint(os.Stderr, "Enter secret: ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("reading input: %w", err)
		}
		secret = []byte(strings.TrimRight(line, "\n"))
	}

	ciphertext, err := age.Encrypt(secret, passphrase)
	if err != nil {
		return err
	}

	if err := s.Write(cfg.Insert.Name, ciphertext); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Secret %q saved.\n", cfg.Insert.Name)
	return nil
}
