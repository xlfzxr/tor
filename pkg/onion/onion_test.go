package onion

import "testing"

func validConfig() Config {
	return Config{
		Nickname: "torstack",
		DataDir:  "/tmp/torstack-hs",
		Ports:    map[int]string{80: "127.0.0.1:8080"},
	}
}

func TestValidateOK(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	multi := validConfig()
	multi.Ports[443] = "127.0.0.1:8443"
	if err := multi.Validate(); err != nil {
		t.Fatalf("Validate multi: %v", err)
	}
}

func TestValidateBad(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{"empty nickname", func(c *Config) { c.Nickname = "" }},
		{"blank nickname", func(c *Config) { c.Nickname = "  " }},
		{"empty datadir", func(c *Config) { c.DataDir = "" }},
		{"no ports", func(c *Config) { c.Ports = nil }},
		{"empty ports", func(c *Config) { c.Ports = map[int]string{} }},
		{"port zero", func(c *Config) { c.Ports = map[int]string{0: "127.0.0.1:80"} }},
		{"port too big", func(c *Config) { c.Ports = map[int]string{70000: "127.0.0.1:80"} }},
		{"empty target", func(c *Config) { c.Ports = map[int]string{80: "" } }},
		{"blank target", func(c *Config) { c.Ports = map[int]string{80: "  "} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validConfig()
			tc.mut(&c)
			if err := c.Validate(); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
