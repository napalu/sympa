package totp

import (
	"testing"
)

func TestGenerate(t *testing.T) {
	// Just verify it returns a 6-digit code without error
	code, err := Generate("JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("code length = %d, want 6", len(code))
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("code contains non-digit: %c", c)
		}
	}
}

func TestGenerateWithSpaces(t *testing.T) {
	// Keys are sometimes displayed with spaces
	code, err := Generate("JBSW Y3DP EHPK 3PXP")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("code length = %d, want 6", len(code))
	}
}

func TestGenerateInvalidKey(t *testing.T) {
	_, err := Generate("!!!invalid!!!")
	if err == nil {
		t.Error("expected error for invalid key")
	}
}
