package fields

import (
	"testing"
)

func TestParse(t *testing.T) {
	data := []byte("hunter2\nuser: florent\ntotp: JBSWY3DPEHPK3PXP\nurl: https://gmail.com\nrecovery: ABCD-1234\nrecovery: EFGH-5678")
	s := Parse(data)

	if s.Password != "hunter2" {
		t.Errorf("password = %q, want %q", s.Password, "hunter2")
	}
	if v, ok := s.Get("user"); !ok || v != "florent" {
		t.Errorf("Get(user) = %q, %v", v, ok)
	}
	if v, ok := s.Get("URL"); !ok || v != "https://gmail.com" {
		t.Errorf("Get(URL) = %q, %v (case-insensitive)", v, ok)
	}
	if v, ok := s.Get("totp"); !ok || v != "JBSWY3DPEHPK3PXP" {
		t.Errorf("Get(totp) = %q, %v", v, ok)
	}
	recoveries := s.GetAll("recovery")
	if len(recoveries) != 2 || recoveries[0] != "ABCD-1234" || recoveries[1] != "EFGH-5678" {
		t.Errorf("GetAll(recovery) = %v", recoveries)
	}
	if _, ok := s.Get("nonexistent"); ok {
		t.Error("Get(nonexistent) should return false")
	}
}

func TestParsePasswordOnly(t *testing.T) {
	s := Parse([]byte("just-a-password"))
	if s.Password != "just-a-password" {
		t.Errorf("password = %q", s.Password)
	}
	if len(s.Fields) != 0 {
		t.Errorf("expected no fields, got %d", len(s.Fields))
	}
}

func TestParseMultilineBlankTerminator(t *testing.T) {
	data := []byte("pass\nprivkey: |\n-----BEGIN KEY-----\nAAAABBBB\n-----END KEY-----\n\nurl: https://example.com")
	s := Parse(data)

	want := "-----BEGIN KEY-----\nAAAABBBB\n-----END KEY-----"
	if v, ok := s.Get("privkey"); !ok || v != want {
		t.Errorf("Get(privkey) = %q, %v; want %q", v, ok, want)
	}
	if v, ok := s.Get("url"); !ok || v != "https://example.com" {
		t.Errorf("Get(url) = %q, %v", v, ok)
	}
}

func TestParseMultilineFieldTerminator(t *testing.T) {
	data := []byte("pass\ncert: |\nline1\nline2\nuser: florent")
	s := Parse(data)

	if v, ok := s.Get("cert"); !ok || v != "line1\nline2" {
		t.Errorf("Get(cert) = %q, %v", v, ok)
	}
	if v, ok := s.Get("user"); !ok || v != "florent" {
		t.Errorf("Get(user) = %q, %v", v, ok)
	}
}

func TestParseMultilineEOF(t *testing.T) {
	data := []byte("pass\nnotes: |\nfirst line\nsecond line")
	s := Parse(data)

	if v, ok := s.Get("notes"); !ok || v != "first line\nsecond line" {
		t.Errorf("Get(notes) = %q, %v", v, ok)
	}
}

func TestParseMultilineEmpty(t *testing.T) {
	data := []byte("pass\nempty: |\n\nuser: florent")
	s := Parse(data)

	if v, ok := s.Get("empty"); !ok || v != "" {
		t.Errorf("Get(empty) = %q, want empty string", v)
	}
	if v, ok := s.Get("user"); !ok || v != "florent" {
		t.Errorf("Get(user) = %q, %v", v, ok)
	}
}

func TestParseMultilineMixed(t *testing.T) {
	data := []byte("pass\nuser: florent\nkey: |\nAAA\nBBB\n\ntotp: JBSWY3DPEHPK3PXP\nnotes: |\nline one\nline two")
	s := Parse(data)

	if v, ok := s.Get("user"); !ok || v != "florent" {
		t.Errorf("Get(user) = %q, %v", v, ok)
	}
	if v, ok := s.Get("key"); !ok || v != "AAA\nBBB" {
		t.Errorf("Get(key) = %q, %v", v, ok)
	}
	if v, ok := s.Get("totp"); !ok || v != "JBSWY3DPEHPK3PXP" {
		t.Errorf("Get(totp) = %q, %v", v, ok)
	}
	if v, ok := s.Get("notes"); !ok || v != "line one\nline two" {
		t.Errorf("Get(notes) = %q, %v", v, ok)
	}
}
