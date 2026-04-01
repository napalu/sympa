package commands

import (
	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

func handleLs(p *goopt.Parser, _ *goopt.Command) error {
	cfg, ok := goopt.GetStructCtxAs[*options.Config](p)
	if !ok {
		return errNoConfig
	}
	s := store.New()
	if err := requireInit(s); err != nil {
		return err
	}
	return s.List(cfg.Ls.Subfolder)
}
