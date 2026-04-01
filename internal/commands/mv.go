package commands

import (
	"fmt"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

func handleMv(p *goopt.Parser, _ *goopt.Command) error {
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

	if !s.Exists(cfg.Mv.Source) {
		return fmt.Errorf("secret %q does not exist", cfg.Mv.Source)
	}
	if s.Exists(cfg.Mv.Dest) && !cfg.Mv.Force {
		return fmt.Errorf("secret %q already exists (use -f to overwrite)", cfg.Mv.Dest)
	}

	return s.Rename(cfg.Mv.Source, cfg.Mv.Dest)
}
