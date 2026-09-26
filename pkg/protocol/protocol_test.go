package protocol

import (
	"net"
	"testing"
	"time"
)

func TestVersionsRoundtrip(t *testing.T) {
	want := []uint16{3, 4, 5}
	b := VersionsPayload(want...)
	got, err := ParseVersions(b)
	if err != nil {
		t.Fatalf("ParseVersions: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("versions[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestParseVersionsBad(t *testing.T) {
	for _, b := range [][]byte{nil, {}, {0x00}, {0x00, 0x03, 0x00}} {
		if _, err := ParseVersions(b); err == nil {
			t.Fatalf("len %d: expected error", len(b))
		}
	}
}

func TestNegotiateVersion(t *testing.T) {
	v, err := NegotiateVersion([]uint16{3, 4, 5}, []uint16{4, 5, 6})
	if err != nil {
		t.Fatalf("Negotiate: %v", err)
	}
	if v != 5 {
		t.Fatalf("negotiated = %d, want 5", v)
	}
	if _, err := NegotiateVersion([]uint16{3}, []uint16{4}); err == nil {
		t.Fatalf("expected no-common-version error")
	}
}

func TestNetinfoRoundtripIPv4(t *testing.T) {
	n := Netinfo{
		Timestamp: uint32(time.Now().Unix()),
		Receiver:  net.ParseIP("1.2.3.4"),
		Senders:   []net.IP{net.ParseIP("5.6.7.8"), net.ParseIP("9.10.11.12")},
	}
	b, err := MarshalNetinfo(n)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := ParseNetinfo(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Timestamp != n.Timestamp {
		t.Fatalf("timestamp = %d, want %d", got.Timestamp, n.Timestamp)
	}
	if !got.Receiver.Equal(n.Receiver) {
		t.Fatalf("receiver = %v, want %v", got.Receiver, n.Receiver)
	}
	if len(got.Senders) != len(n.Senders) {
		t.Fatalf("senders len = %d, want %d", len(got.Senders), len(n.Senders))
	}
	for i := range n.Senders {
		if !got.Senders[i].Equal(n.Senders[i]) {
			t.Fatalf("sender[%d] = %v, want %v", i, got.Senders[i], n.Senders[i])
		}
	}
}

func TestNetinfoRoundtripIPv6(t *testing.T) {
	n := Netinfo{
		Timestamp: 1234567890,
		Receiver:  net.ParseIP("::1"),
		Senders:   []net.IP{net.ParseIP("2001:db8::1")},
	}
	b, err := MarshalNetinfo(n)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := ParseNetinfo(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !got.Receiver.Equal(n.Receiver) {
		t.Fatalf("receiver = %v, want %v", got.Receiver, n.Receiver)
	}
	if len(got.Senders) != 1 || !got.Senders[0].Equal(n.Senders[0]) {
		t.Fatalf("senders mismatch: %v", got.Senders)
	}
}

func TestNetinfoBad(t *testing.T) {
	if _, err := ParseNetinfo(nil); err == nil {
		t.Fatalf("expected error for nil")
	}
	if _, err := ParseNetinfo([]byte{0, 0}); err == nil {
		t.Fatalf("expected error for truncated")
	}
	n := Netinfo{Timestamp: 1, Receiver: net.ParseIP("1.2.3.4")}
	b, err := MarshalNetinfo(n)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if _, err := ParseNetinfo(append(b, 0xFF)); err == nil {
		t.Fatalf("expected trailing-bytes error")
	}
	if _, err := MarshalNetinfo(Netinfo{Timestamp: 1}); err == nil {
		t.Fatalf("expected error for nil receiver IP")
	}
}

func TestClientConfigDefaults(t *testing.T) {
	cfg := ClientConfig("example.com")
	if cfg.ServerName != "example.com" {
		t.Fatalf("ServerName = %q", cfg.ServerName)
	}
	if cfg.InsecureSkipVerify {
		t.Fatalf("InsecureSkipVerify must be false")
	}
	if DialTimeout <= 0 {
		t.Fatalf("DialTimeout must be positive")
	}
}
