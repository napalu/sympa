package secret

import "os"

// ramDir returns a RAM-backed directory and an optional cleanup function.
// On Linux, /dev/shm is a tmpfs mounted by default.
func ramDir() (string, func()) {
	info, err := os.Stat("/dev/shm")
	if err == nil && info.IsDir() {
		return "/dev/shm", nil
	}
	return "", nil
}
