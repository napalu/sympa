//go:build linux

package agent

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// harden applies security hardening to the agent process on Linux:
// - mlockall: prevent memory from being swapped to disk
// - RLIMIT_CORE=0: disable core dumps
// - PR_SET_DUMPABLE=0: prevent /proc/pid/mem reads by same user
func harden() {
	if err := unix.Mlockall(unix.MCL_CURRENT | unix.MCL_FUTURE); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: mlockall unavailable (%v)\n", err)
	}

	_ = syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{Cur: 0, Max: 0})

	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, unix.PR_SET_DUMPABLE, 0, 0); errno != 0 {
		fmt.Fprintf(os.Stderr, "Warning: PR_SET_DUMPABLE failed (%v)\n", errno)
	}
}
