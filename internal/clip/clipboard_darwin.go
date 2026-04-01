package clip

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// Copy writes text to the macOS clipboard via pbcopy.
func Copy(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	return nil
}

// Paste reads text from the macOS clipboard via pbpaste.
func Paste() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return string(out), nil
}

func detachProc() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
