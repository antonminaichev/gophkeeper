package client

import "testing"

func TestDebugString_AliasPreferred(t *testing.T) {
	ci := &CachedItem{
		ID:      "12345678-aaaa-bbbb-cccc-ddddeeeeffff",
		Alias:   "myalias",
		Type:    "TEXT",
		Version: 3,
		Meta:    map[string]string{"title": "Hello"},
	}
	got := ci.DebugString()
	wantPrefix := "@myalias"
	if got[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("prefix = %q, want %q; full=%q", got[:len(wantPrefix)], wantPrefix, got)
	}
	if want := "TEXT"; !contains(got, want) {
		t.Fatalf("type missing: %q", got)
	}
	if want := "3"; !contains(got, want) {
		t.Fatalf("version missing: %q", got)
	}
	if want := "Hello"; !contains(got, want) {
		t.Fatalf("title missing: %q", got)
	}
}

func TestDebugString_HumanIDFallback(t *testing.T) {
	ci := &CachedItem{
		ID:      "abcd1234",
		HumanID: 42,
		Type:    "CARD",
		Version: 1,
		Meta:    map[string]string{"title": "T"},
	}
	got := ci.DebugString()
	if want := "#42"; !contains(got, want) {
		t.Fatalf("id column want %q, got %q", want, got)
	}
}

func TestDebugString_UUIDShortFallback(t *testing.T) {
	ci := &CachedItem{
		ID:      "1234567", // меньше 8 символов — берётся как есть
		Type:    "BINARY",
		Version: 2,
		Meta:    map[string]string{"title": "X"},
	}
	got := ci.DebugString()
	if want := "1234567"; !contains(got, want) {
		t.Fatalf("short id want %q, got %q", want, got)
	}

	ci.ID = "12345678ZZZ" // >=8 — берутся первые 8
	got = ci.DebugString()
	if want := "12345678"; !contains(got, want) {
		t.Fatalf("shortened id want %q, got %q", want, got)
	}
}

func TestDebugString_NoTitle(t *testing.T) {
	ci := &CachedItem{
		ID:      "ffffffff",
		Type:    "TEXT",
		Version: 10,
		Meta:    map[string]string{"title": "   "}, // пустой после TrimSpace
	}
	got := ci.DebugString()
	if want := "(no title)"; !contains(got, want) {
		t.Fatalf("default title want %q, got %q", want, got)
	}
}

// small helper
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && (indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if s[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
