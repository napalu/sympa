package commands

import (
	"fmt"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/clip"
	"github.com/napalu/sympa/internal/fields"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
	"github.com/napalu/sympa/internal/totp"
)

func handleTotp(p *goopt.Parser, _ *goopt.Command) error {
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
	if !s.Exists(cfg.Totp.Name) {
		return fmt.Errorf("secret %q does not exist", cfg.Totp.Name)
	}

	ciphertext, err := s.Read(cfg.Totp.Name)
	if err != nil {
		return err
	}

	plaintext, _, err := cachedDecrypt(ciphertext, cfg.Totp.Name, "Enter passphrase: ", cfg.Keyfile)
	if err != nil {
		return err
	}

	secret := fields.Parse(plaintext)
	totpKey, ok := secret.Get("totp")
	if !ok {
		return fmt.Errorf("no totp field found in secret %q", cfg.Totp.Name)
	}

	code, err := totp.Generate(totpKey)
	if err != nil {
		return err
	}

	if cfg.Totp.Clip {
		return clip.CopyAndClear(code, clip.ClearTimeout)
	}

	fmt.Println(code)
	return nil
}
