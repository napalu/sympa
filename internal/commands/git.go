package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/store"
)

func handleGit(p *goopt.Parser, _ *goopt.Command) error {
	s := store.New()
	if err := requireInit(s); err != nil {
		return err
	}

	// greedy:true on the Git command causes goopt to collect everything after
	// "git" as unbound positional args instead of trying to match them as
	// commands. Extract them for the git invocation.
	var gitArgs []string
	for _, pos := range p.GetPositionalArgs() {
		if pos.Argument == nil {
			gitArgs = append(gitArgs, pos.Value)
		}
	}
	if len(gitArgs) == 0 {
		return fmt.Errorf("no git command specified (e.g., sympa git init)")
	}

	args := append([]string{"-C", s.Dir()}, gitArgs...)
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("git: %w", err)
	}
	return nil
}
