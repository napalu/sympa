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
