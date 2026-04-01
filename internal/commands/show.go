package commands

import (
	"fmt"
	"os"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/clip"
	"github.com/napalu/sympa/internal/fields"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

func handleShow(p *goopt.Parser, _ *goopt.Command) error {
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
	if !s.Exists(cfg.Show.Name) {
		return fmt.Errorf("secret %q does not exist", cfg.Show.Name)
	}

	ciphertext, err := s.Read(cfg.Show.Name)
	if err != nil {
		return err
	}

	plaintext, _, err := cachedDecrypt(ciphertext, cfg.Show.Name, "Enter passphrase: ", cfg.Keyfile)
	if err != nil {
		return err
	}

	secret := fields.Parse(plaintext)

	// Determine what to output
	var output string
	switch {
	case cfg.Show.Field != "":
		val, ok := secret.Get(cfg.Show.Field)
		if !ok {
			return fmt.Errorf("field %q not found in secret %q", cfg.Show.Field, cfg.Show.Name)
		}
		output = val
	default:
		output = secret.Password
	}

	if cfg.Show.Clip {
		return clip.CopyAndClear(output, clip.ClearTimeout)
	}

	// No -c and no -f: print everything as before
	if cfg.Show.Field == "" {
		os.Stdout.Write(plaintext)
		if len(plaintext) > 0 && plaintext[len(plaintext)-1] != '\n' {
			fmt.Println()
		}
		return nil
	}

	// -f without -c: print just the field value
	fmt.Println(output)
	return nil
}
