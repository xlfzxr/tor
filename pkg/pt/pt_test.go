package pt

import "testing"

func TestKnown(t *testing.T) {
	k := Known()
	if len(k) != 3 {
		t.Fatalf("Known len = %d, want 3", len(k))
	}
	want := map[string]bool{"obfs4": true, "snowflake": true, "webtunnel": true}
	for _, n := range k {
		if !want[n] {
			t.Fatalf("unexpected transport %q", n)
		}
	}
}

func TestValidateOK(t *testing.T) {
	for _, name := range Known() {
		tr := Transport{Name: name}
		if err := tr.Validate(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// Case-insensitive + args allowed.
	tr := Transport{Name: "OBFS4", Args: []string{"cert=abc", "iat-mode=0"}}
	if err := tr.Validate(); err != nil {
		t.Fatalf("case-insensitive: %v", err)
	}
}

func TestValidateBad(t *testing.T) {
	for _, tr := range []Transport{
		{},
		{Name: "  "},
		{Name: "meek"},
		{Name: "obfs4", Args: []string{""}},
		{Name: "obfs4", Args: []string{"ok", "  "}},
	} {
		if err := tr.Validate(); err == nil {
			t.Fatalf("%+v: expected error", tr)
		}
	}
}
