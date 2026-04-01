package secret

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const ramdiskSectors = 4096 // 4096 × 512 bytes = 2 MB

func ramdiskLabel() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("sympa-%d-%s", os.Getpid(), hex.EncodeToString(b))
}

// ramDir creates a RAM disk on macOS via hdiutil and returns its mount point
// along with a cleanup function that detaches it.
func ramDir() (string, func()) {
	label := ramdiskLabel()

	dev, err := createRAMDisk()
	if err != nil {
		return "", nil
	}

	mountPoint := "/Volumes/" + label
	if err := formatRAMDisk(dev, label); err != nil {
		detachRAMDisk(dev)
		return "", nil
	}

	// Restrict access to current user only
	os.Chmod(mountPoint, 0700)

	return mountPoint, func() {
		detachRAMDisk(dev)
	}
}

func createRAMDisk() (string, error) {
	cmd := exec.Command("hdiutil", "attach", "-nomount",
		fmt.Sprintf("ram://%d", ramdiskSectors))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("creating RAM disk: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func formatRAMDisk(dev, label string) error {
	cmd := exec.Command("diskutil", "eraseVolume",
		"HFS+", label, dev)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("formatting RAM disk: %w", err)
	}
	return nil
}

func detachRAMDisk(dev string) {
	exec.Command("hdiutil", "detach", dev, "-force").Run()
}
