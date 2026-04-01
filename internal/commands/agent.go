package commands

import (
	"fmt"
	"os"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/agent"
)

func handleAgentStart(_ *goopt.Parser, _ *goopt.Command) error {
	srv := agent.NewServer(agent.TTL())
	return srv.Run(agent.SocketPath())
}

func handleAgentStop(_ *goopt.Parser, _ *goopt.Command) error {
	c := agent.NewClient()
	if !c.Ping() {
		fmt.Fprintln(os.Stderr, "Agent is not running.")
		return nil
	}
	if err := c.Shutdown(); err != nil {
		return fmt.Errorf("stopping agent: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Agent stopped.")
	return nil
}

func handleAgentStatus(_ *goopt.Parser, _ *goopt.Command) error {
	c := agent.NewClient()
	if c.Ping() {
		mode := "r"
		if agent.WriteEnabled() {
			mode = "rw"
		}
		fmt.Printf("Agent is running.\n")
		fmt.Printf("Socket: %s\n", agent.SocketPath())
		fmt.Printf("TTL: %s\n", agent.TTL())
		fmt.Printf("Mode: %s\n", mode)
	} else {
		fmt.Println("Agent is not running.")
	}
	return nil
}
