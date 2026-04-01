//go:build darwin

package agent

import (
	"syscall"
)

// harden applies security hardening to the agent process on macOS:
// - RLIMIT_CORE=0: disable core dumps
// mlockall is not available on macOS.
func harden() {
	_ = syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{Cur: 0, Max: 0})
}
