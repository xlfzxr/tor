package protocol

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// Tor uses TLS for link encryption (tor-spec: handshake + renegotiation).
// This wraps stdlib TLS with sane defaults for relay connections.
func ClientConfig(serverName string) *tls.Config {
	return &tls.Config{
		ServerName:         serverName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}
}

// VersionsCell is the link-protocol negotiation payload (link v3+).
func VersionsPayload(versions ...uint16) []byte {
	out := make([]byte, 0, len(versions)*2)
	for _, v := range versions {
		out = append(out, byte(v>>8), byte(v))
	}
	return out
}

// ParseVersions decodes a VERSIONS cell payload into version numbers.
// Payload must be a non-empty even-length big-endian u16 list.
func ParseVersions(b []byte) ([]uint16, error) {
	if len(b) == 0 || len(b)%2 != 0 {
		return nil, fmt.Errorf("bad versions payload len %d", len(b))
	}
	out := make([]uint16, 0, len(b)/2)
	for i := 0; i < len(b); i += 2 {
		out = append(out, binary.BigEndian.Uint16(b[i:i+2]))
	}
	return out, nil
}

// NegotiateVersion picks the highest version present in both lists.
// Returns an error when there is no common version.
func NegotiateVersion(ours, theirs []uint16) (uint16, error) {
	set := make(map[uint16]bool, len(theirs))
	for _, v := range theirs {
		set[v] = true
	}
	var best uint16
	found := false
	for _, v := range ours {
		if set[v] && (!found || v > best) {
			best = v
			found = true
		}
	}
	if !found {
		return 0, fmt.Errorf("no common link version")
	}
	return best, nil
}

// Netinfo is the link handshake address attestation (tor-spec 4.5).
type Netinfo struct {
	Timestamp uint32
	Receiver  net.IP
	Senders   []net.IP
}

// MarshalNetinfo encodes Timestamp + receiver + sender addresses.
// Address encoding: ATYPE(1) + ALEN(1) + ADDR, ATYPE 4=IPv4, 6=IPv6.
func MarshalNetinfo(n Netinfo) ([]byte, error) {
	enc, err := encodeAddr(n.Receiver)
	if err != nil {
		return nil, fmt.Errorf("receiver: %w", err)
	}
	if len(n.Senders) > 255 {
		return nil, fmt.Errorf("too many sender addresses: %d", len(n.Senders))
	}
	out := make([]byte, 4, 4+len(enc)+1+len(n.Senders)*18)
	binary.BigEndian.PutUint32(out[0:4], n.Timestamp)
	out = append(out, enc...)
	out = append(out, byte(len(n.Senders)))
	for _, s := range n.Senders {
		e, err := encodeAddr(s)
		if err != nil {
			return nil, fmt.Errorf("sender: %w", err)
		}
		out = append(out, e...)
	}
	return out, nil
}

// ParseNetinfo decodes a NETINFO cell payload.
func ParseNetinfo(b []byte) (Netinfo, error) {
	var n Netinfo
	if len(b) < 4+3 {
		return n, fmt.Errorf("netinfo too short: %d", len(b))
	}
	n.Timestamp = binary.BigEndian.Uint32(b[0:4])
	rest := b[4:]
	ip, consumed, err := decodeAddr(rest)
	if err != nil {
		return n, fmt.Errorf("receiver: %w", err)
	}
	n.Receiver = ip
	rest = rest[consumed:]
	if len(rest) < 1 {
		return n, fmt.Errorf("netinfo missing sender count")
	}
	count := int(rest[0])
	rest = rest[1:]
	n.Senders = make([]net.IP, 0, count)
	for i := 0; i < count; i++ {
		ip, consumed, err := decodeAddr(rest)
		if err != nil {
			return n, fmt.Errorf("sender %d: %w", i, err)
		}
		n.Senders = append(n.Senders, ip)
		rest = rest[consumed:]
	}
	if len(rest) != 0 {
		return n, fmt.Errorf("netinfo trailing %d bytes", len(rest))
	}
	return n, nil
}

func encodeAddr(ip net.IP) ([]byte, error) {
	if v4 := ip.To4(); v4 != nil {
		return append([]byte{4, 4}, v4...), nil
	}
	if v6 := ip.To16(); v6 != nil {
		return append([]byte{6, 16}, v6...), nil
	}
	return nil, fmt.Errorf("unparseable IP %q", ip.String())
}

func decodeAddr(b []byte) (net.IP, int, error) {
	if len(b) < 2 {
		return nil, 0, fmt.Errorf("addr header too short")
	}
	typ, ln := b[0], int(b[1])
	switch typ {
	case 4:
		if ln != 4 {
			return nil, 0, fmt.Errorf("bad IPv4 len %d", ln)
		}
	case 6:
		if ln != 16 {
			return nil, 0, fmt.Errorf("bad IPv6 len %d", ln)
		}
	default:
		return nil, 0, fmt.Errorf("unsupported addr type %d", typ)
	}
	if len(b) < 2+ln {
		return nil, 0, fmt.Errorf("addr truncated: need %d, have %d", 2+ln, len(b))
	}
	return net.IP(append([]byte(nil), b[2:2+ln]...)), 2 + ln, nil
}

const DialTimeout = 15 * time.Second
