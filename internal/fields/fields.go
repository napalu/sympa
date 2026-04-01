package fields

import (
	"strings"
)

// Field is a key-value pair from a secret.
type Field struct {
	Key   string
	Value string
}

// Secret represents a parsed secret with a password and optional fields.
type Secret struct {
	Password string
	Fields   []Field
}

// Parse parses plaintext into a Secret.
// Line 1 is the password. Subsequent lines in "key: value" format become fields.
// Lines without a colon are ignored.
func Parse(plaintext []byte) *Secret {
	lines := strings.Split(string(plaintext), "\n")
	s := &Secret{}

	if len(lines) > 0 {
		s.Password = lines[0]
	}

	for _, line := range lines[1:] {
		key, value, ok := parseField(line)
		if ok {
			s.Fields = append(s.Fields, Field{Key: key, Value: value})
		}
	}

	return s
}

// Get returns the value of the first field matching key (case-insensitive).
func (s *Secret) Get(key string) (string, bool) {
	lower := strings.ToLower(key)
	for _, f := range s.Fields {
		if strings.ToLower(f.Key) == lower {
			return f.Value, true
		}
	}
	return "", false
}

// GetAll returns all values for fields matching key (case-insensitive).
func (s *Secret) GetAll(key string) []string {
	lower := strings.ToLower(key)
	var values []string
	for _, f := range s.Fields {
		if strings.ToLower(f.Key) == lower {
			values = append(values, f.Value)
		}
	}
	return values
}

func parseField(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 1 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}
