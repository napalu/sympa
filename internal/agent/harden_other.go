//go:build !linux && !darwin

package agent

// harden is a no-op on unsupported platforms.
func harden() {}
