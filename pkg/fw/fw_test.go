package fw

import "testing"

func TestDefaultDeniesAbusePorts(t *testing.T) {
	d := Default()
	for _, p := range []string{"25", "119", "445", "563", "6699"} {
		if d.Allows(p) {
			t.Fatalf("default should deny %s", p)
		}
	}
	for _, p := range []string{"80", "443", "9050", "22"} {
		if !d.Allows(p) {
			t.Fatalf("default should allow %s", p)
		}
	}
}

func TestRangeDeny(t *testing.T) {
	d := Default()
	// 135-139 range from default policy.
	for _, p := range []string{"135", "136", "139"} {
		if d.Allows(p) {
			t.Fatalf("should deny ranged %s", p)
		}
	}
	if !d.Allows("140") {
		t.Fatalf("140 just outside 135-139 should be allowed")
	}
	// 6881-6999 torrent range.
	if d.Allows("6881") || d.Allows("6999") {
		t.Fatalf("torrent range should be denied")
	}
	// Substring must NOT match: deny "25" must not block "125" or "2525".
	p := Policy{DenyPorts: []string{"25"}}
	if !p.Allows("125") {
		t.Fatalf("125 should be allowed (no substring match)")
	}
	if !p.Allows("2525") {
		t.Fatalf("2525 should be allowed (no substring match)")
	}
	if p.Allows("25") {
		t.Fatalf("25 should be denied")
	}
}

func TestAllowListRestrictive(t *testing.T) {
	p := Policy{AllowPorts: []string{"443", "80"}}
	if !p.Allows("443") || !p.Allows("80") {
		t.Fatalf("explicit allow should pass")
	}
	if p.Allows("22") {
		t.Fatalf("not in allow list should be denied")
	}
	// Deny wins over allow.
	p2 := Policy{AllowPorts: []string{"443"}, DenyPorts: []string{"443"}}
	if p2.Allows("443") {
		t.Fatalf("deny must win over allow")
	}
	// Allow ranges work too.
	p3 := Policy{AllowPorts: []string{"8000-8100"}}
	if !p3.Allows("8050") {
		t.Fatalf("8050 in allow range should pass")
	}
	if p3.Allows("8110") {
		t.Fatalf("8110 outside allow range should fail")
	}
}

func TestValidate(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("Default Validate: %v", err)
	}
	if err := (Policy{DenyPorts: []string{"nope"}}).Validate(); err == nil {
		t.Fatalf("expected error for bad entry")
	}
	if err := (Policy{AllowPorts: []string{"0"}}).Validate(); err == nil {
		t.Fatalf("expected error for port 0")
	}
	if err := (Policy{DenyPorts: []string{"139-135"}}).Validate(); err == nil {
		t.Fatalf("expected error for inverted range")
	}
}
