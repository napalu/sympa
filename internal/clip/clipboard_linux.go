package clip

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Copy writes text to the clipboard, trying Wayland first then X11.
func Copy(text string) error {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if path, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command(path)
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command(path, "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	if path, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command(path, "--clipboard", "--input")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")
}

// Paste reads text from the clipboard, trying Wayland first then X11.
func Paste() (string, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if path, err := exec.LookPath("wl-paste"); err == nil {
			out, err := exec.Command(path, "--no-newline").Output()
			return string(out), err
		}
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		out, err := exec.Command(path, "-selection", "clipboard", "-o").Output()
		return string(out), err
	}
	if path, err := exec.LookPath("xsel"); err == nil {
		out, err := exec.Command(path, "--clipboard", "--output").Output()
		return string(out), err
	}
	return "", fmt.Errorf("no clipboard tool found")
}

func detachProc() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
