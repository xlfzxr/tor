package fw

import (
	"fmt"
	"strconv"
	"strings"
)

// Policy is a minimal egress firewall model (which ports may exit).
// Termux non-root cannot enforce via iptables, so this is advisory:
// doctor + client warn when ExitPolicy would leak.
type Policy struct {
	AllowPorts []string // e.g. ["443", "80"]
	DenyPorts  []string // e.g. ["25", "119"]
}

// Allows reports whether port is permitted.
// Deny list wins over allow list; entries may be "443" or "135-139".
func (p Policy) Allows(port string) bool {
	for _, d := range p.DenyPorts {
		if matchPort(d, port) {
			return false
		}
	}
	if len(p.AllowPorts) == 0 {
		return true
	}
	for _, a := range p.AllowPorts {
		if matchPort(a, port) {
			return true
		}
	}
	return false
}

// Validate checks every entry parses as a port or port range.
func (p Policy) Validate() error {
	for _, e := range append(append([]string{}, p.AllowPorts...), p.DenyPorts...) {
		if _, _, err := parseRange(e); err != nil {
			return fmt.Errorf("bad port entry %q: %w", e, err)
		}
	}
	return nil
}

// matchPort reports whether entry ("443" or "135-139") covers port.
func matchPort(entry, port string) bool {
	elo, ehi, err := parseRange(strings.TrimSpace(entry))
	if err != nil {
		return strings.TrimSpace(entry) == strings.TrimSpace(port)
	}
	v, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil {
		return false
	}
	return v >= elo && v <= ehi
}

func parseRange(s string) (int, int, error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		lo, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, err
		}
		hi, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, err
		}
		if lo < 1 || hi > 65535 || lo > hi {
			return 0, 0, fmt.Errorf("range %d-%d out of bounds", lo, hi)
		}
		return lo, hi, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, 0, err
	}
	if v < 1 || v > 65535 {
		return 0, 0, fmt.Errorf("port %d out of range", v)
	}
	return v, v, nil
}

// Default mirrors the mentoreth2-style exit policy (no SMTP/abuse ports).
func Default() Policy {
	return Policy{DenyPorts: []string{"25", "119", "135-139", "445", "563", "1214", "4661-4666", "6346-6429", "6699", "6881-6999"}}
}
