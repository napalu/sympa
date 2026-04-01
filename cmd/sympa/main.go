package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/napalu/sympa/internal/clip"
	"github.com/napalu/sympa/internal/commands"
	"github.com/napalu/sympa/internal/options"
)

// Set by goreleaser ldflags at build time.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	// Internal command: clear clipboard after delay if content unchanged.
	// Invoked by CopyAndClear as a detached background process.
	// Fingerprint is read from stdin to avoid exposing it in process args.
	if len(os.Args) == 3 && os.Args[1] == "_clear-clipboard" {
		secs, err := strconv.Atoi(os.Args[2])
		if err != nil {
			os.Exit(1)
		}
		raw, err := io.ReadAll(os.Stdin)
		if err != nil || len(raw) == 0 {
			os.Exit(1)
		}
		clip.ClearIfMatch(strings.TrimSpace(string(raw)), time.Duration(secs)*time.Second)
		return
	}

	// Hidden command: output completion candidates for shell scripts.
	// Bypasses the parser for speed — completion must feel instant.
	if len(os.Args) >= 3 && os.Args[1] == "__complete" {
		commands.HandleComplete(os.Args[2:])
		return
	}

	cfg := &options.Config{}
	commands.AssignCallbacks(cfg)

	parser, err := options.New(cfg, version, commit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !parser.Parse(os.Args) {
		for _, e := range parser.GetErrors() {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		}
		os.Exit(1)
	}

	if errCount := parser.ExecuteCommands(); errCount > 0 {
		for _, kv := range parser.GetCommandExecutionErrors() {
			fmt.Fprintf(os.Stderr, "Error: %v\n", kv.Value)
		}
		os.Exit(1)
	}
}
