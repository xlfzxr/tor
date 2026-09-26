package pt

import (
	"fmt"
	"strings"
)

// Transport is a pluggable-transport (obfs4/snowflake/webtunnel) descriptor.
// Execution stays out-of-process for now; this only models config.
type Transport struct {
	Name string
	Args []string
}

// Known lists transports torstack can reference in torrc.
func Known() []string { return []string{"obfs4", "snowflake", "webtunnel"} }

// Validate checks the transport name is known and args are well-formed.
func (t Transport) Validate() error {
	name := strings.ToLower(strings.TrimSpace(t.Name))
	if name == "" {
		return fmt.Errorf("transport name required")
	}
	known := false
	for _, k := range Known() {
		if k == name {
			known = true
			break
		}
	}
	if !known {
		return fmt.Errorf("unknown transport %q (known: obfs4, snowflake, webtunnel)", t.Name)
	}
	for i, a := range t.Args {
		if strings.TrimSpace(a) == "" {
			return fmt.Errorf("arg[%d] empty", i)
		}
	}
	return nil
}
