package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Agent env vars are read via os.Getenv rather than goopt struct fields because
// they are global configuration — any command that prompts for a passphrase checks
// the agent, not just the "agent" subcommand. Declaring them as goopt flags would
// scope them to a single command and clutter the help output.
const (
	envAgent      = "SYMPA_AGENT"
	envSock       = "SYMPA_AGENT_SOCK"
	envTimeout    = "SYMPA_AGENT_TIMEOUT"
	envMode       = "SYMPA_AGENT_MODE"
	defaultSocket = "sympa/agent.sock"
	defaultTTL    = 2 * time.Minute
)

// Enabled reports whether the agent is enabled.
// Set SYMPA_AGENT=off to disable auto-spawn and all caching.
func Enabled() bool {
	return !strings.EqualFold(os.Getenv(envAgent), "off")
}

// SocketPath returns the agent Unix domain socket path.
// Resolution: $SYMPA_AGENT_SOCK > $XDG_RUNTIME_DIR/sympa/agent.sock > /tmp/sympa-<uid>/agent.sock
func SocketPath() string {
	if p := os.Getenv(envSock); p != "" {
		return p
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, defaultSocket)
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("sympa-%d", os.Getuid()), "agent.sock")
}

// WriteEnabled reports whether the agent caches write operations.
// Set SYMPA_AGENT_MODE=rw to enable.
func WriteEnabled() bool {
	return os.Getenv(envMode) == "rw"
}

// TTL returns the cache entry lifetime from $SYMPA_AGENT_TIMEOUT (Go duration string).
// Defaults to 2 minutes.
func TTL() time.Duration {
	if s := os.Getenv(envTimeout); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			return d
		}
	}
	return defaultTTL
}
