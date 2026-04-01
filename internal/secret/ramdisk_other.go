//go:build !linux && !darwin

package secret

// ramDir returns empty on unsupported platforms, triggering the fallback.
func ramDir() (string, func()) {
	return "", nil
}
