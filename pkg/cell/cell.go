package cell

import (
	"encoding/binary"
	"fmt"
)

// Tor v1 fixed cell: CircID(2) + CMD(1) + PAYLOAD(509) on old links,
// CircID(4) on v4+ links. We support both for forward-compat.
const (
	V1Len = 512
	V4Len = 514
)

// Commands (subset, tor-spec).
const (
	CmdPadding     = 0
	CmdCreate      = 1
	CmdCreated     = 2
	CmdRelay       = 3
	CmdDestroy     = 4
	CmdCreateFast  = 5
	CmdCreatedFast = 6
	CmdVersions    = 7
	CmdNetinfo     = 8
	CmdRelayEarly  = 9
	CmdCreate2     = 10
	CmdCreated2    = 11
)

// Cell is a decoded Tor link cell.
type Cell struct {
	CircID  uint32
	Command byte
	Payload []byte
	WideID  bool // true = 4-byte CircID (link v4+)
}

// Encode serializes to wire bytes.
func Encode(c Cell) ([]byte, error) {
	plen := len(c.Payload)
	if plen > 509 {
		return nil, fmt.Errorf("payload too large: %d", plen)
	}
	if c.WideID {
		out := make([]byte, 4+1+509)
		binary.BigEndian.PutUint32(out[0:4], c.CircID)
		out[4] = c.Command
		copy(out[5:], c.Payload)
		return out, nil
	}
	if c.CircID > 0xFFFF {
		return nil, fmt.Errorf("circid overflow for narrow cell")
	}
	out := make([]byte, 2+1+509)
	binary.BigEndian.PutUint16(out[0:2], uint16(c.CircID))
	out[2] = c.Command
	copy(out[3:], c.Payload)
	return out, nil
}

// Decode parses wire bytes (512 or 514).
func Decode(b []byte) (Cell, error) {
	switch len(b) {
	case V1Len:
		return Cell{
			CircID:  uint32(binary.BigEndian.Uint16(b[0:2])),
			Command: b[2],
			Payload: append([]byte(nil), b[3:]...),
		}, nil
	case V4Len:
		return Cell{
			CircID:  binary.BigEndian.Uint32(b[0:4]),
			Command: b[4],
			Payload: append([]byte(nil), b[5:]...),
			WideID:  true,
		}, nil
	default:
		return Cell{}, fmt.Errorf("bad cell len %d", len(b))
	}
}
