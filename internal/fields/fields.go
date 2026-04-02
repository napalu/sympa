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
// A field with value "|" starts a multiline block that continues until a blank
// line, the next "key: value" line, or EOF. Lines without a colon are ignored.
func Parse(plaintext []byte) *Secret {
	lines := strings.Split(string(plaintext), "\n")
	s := &Secret{}

	if len(lines) > 0 {
		s.Password = lines[0]
	}

	for i := 1; i < len(lines); i++ {
		key, value, ok := parseField(lines[i])
		if !ok {
			continue
		}
		if value == "|" {
			var block []string
			i++
			for i < len(lines) {
				if lines[i] == "" {
					break
				}
				if _, _, isField := parseField(lines[i]); isField {
					i--
					break
				}
				block = append(block, lines[i])
				i++
			}
			s.Fields = append(s.Fields, Field{Key: key, Value: strings.Join(block, "\n")})
		} else {
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
