//go:build !darwin && !linux

package clip

import (
	"fmt"
	"syscall"
)

// Copy is not supported on this platform.
func Copy(_ string) error {
	return fmt.Errorf("clipboard not supported on this platform")
}

// Paste is not supported on this platform.
func Paste() (string, error) {
	return "", fmt.Errorf("clipboard not supported on this platform")
}

func detachProc() *syscall.SysProcAttr {
	return nil
}
