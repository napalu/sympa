package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

func handleRm(p *goopt.Parser, _ *goopt.Command) error {
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

	name := cfg.Rm.Name

	if cfg.Rm.Recursive && s.IsDir(name) {
		if !cfg.Rm.Force {
			fmt.Fprintf(os.Stderr, "Remove directory %q and all its contents? [y/N] ", name)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				fmt.Fprintln(os.Stderr, "Aborted.")
				return nil
			}
		}
		return s.RemoveDir(name)
	}

	if !s.Exists(name) {
		return fmt.Errorf("secret %q does not exist", name)
	}

	if !cfg.Rm.Force {
		fmt.Fprintf(os.Stderr, "Remove secret %q? [y/N] ", name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}
	}

	return s.Remove(name)
}
