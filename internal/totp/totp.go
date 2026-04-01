package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	digits = 6
	period = 30
)

// Generate computes a TOTP code from a base32-encoded secret key.
// Uses RFC 6238 defaults: HMAC-SHA1, 6 digits, 30-second period.
func Generate(secret string) (string, error) {
	// Clean up the secret: remove spaces, uppercase
	secret = strings.ToUpper(strings.ReplaceAll(secret, " ", ""))

	// Pad base32 if needed
	if pad := len(secret) % 8; pad != 0 {
		secret += strings.Repeat("=", 8-pad)
	}

	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("invalid TOTP secret: %w", err)
	}

	counter := uint64(time.Now().Unix()) / period
	return generateCode(key, counter)
}

func generateCode(key []byte, counter uint64) (string, error) {
	// Counter as 8-byte big-endian
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	// HMAC-SHA1
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// Modulo to get the desired number of digits
	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}

	return fmt.Sprintf("%0*d", digits, code%mod), nil
}
