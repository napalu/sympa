package clip

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const ClearTimeout = 45 * time.Second

// CopyAndClear copies text to the clipboard, spawns a detached sympa process
// to clear it after the timeout (only if the clipboard still contains the
// copied text), and returns immediately.
func CopyAndClear(text string, timeout time.Duration) error {
	if err := Copy(text); err != nil {
		return err
	}

	// Read back from clipboard to capture any transformation (e.g. trailing
	// newline) the clipboard tool may apply, so the fingerprint matches what
	// a later Paste() will return.
	stored, err := Paste()
	if err != nil {
		stored = text // fall back to original if paste unavailable
	}
	fingerprint := fmt.Sprintf("%x", sha256.Sum256([]byte(stored)))

	exePath, err := os.Executable()
	if err == nil {
		cmd := exec.Command(exePath, "_clear-clipboard",
			fmt.Sprintf("%d", int(timeout.Seconds())))
		cmd.SysProcAttr = detachProc()
		cmd.Stdout = nil
		cmd.Stderr = nil
		stdin, pipeErr := cmd.StdinPipe()
		if pipeErr == nil {
			if cmd.Start() == nil {
				stdin.Write([]byte(fingerprint))
				stdin.Close()
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Copied to clipboard. Clears in %d seconds.\n", int(timeout.Seconds()))
	return nil
}

// ClearIfMatch waits for the given duration, then clears the clipboard only if
// its current SHA-256 fingerprint matches the expected value.
func ClearIfMatch(fingerprint string, delay time.Duration) {
	time.Sleep(delay)

	current, err := Paste()
	if err != nil {
		return
	}
	currentFP := fmt.Sprintf("%x", sha256.Sum256([]byte(current)))
	if currentFP == fingerprint {
		Copy("")
	}
}
