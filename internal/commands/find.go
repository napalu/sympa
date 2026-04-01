package commands

import (
	"fmt"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

func handleFind(p *goopt.Parser, _ *goopt.Command) error {
	cfg, ok := goopt.GetStructCtxAs[*options.Config](p)
	if !ok {
		return errNoConfig
	}
	s := store.New()
	if err := requireInit(s); err != nil {
		return err
	}

	matches, err := s.Find(cfg.Find.Pattern)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("no secrets matching %q", cfg.Find.Pattern)
	}
	for _, name := range matches {
		fmt.Println(name)
	}
	return nil
}
