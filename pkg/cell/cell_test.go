package cell

import (
	"bytes"
	"testing"
)

func TestRoundtripNarrow512(t *testing.T) {
	payload := []byte("hello-tor-narrow")
	c := Cell{CircID: 0x1234, Command: CmdRelay, Payload: payload}
	b, err := Encode(c)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(b) != V1Len {
		t.Fatalf("narrow len = %d, want %d", len(b), V1Len)
	}
	got, err := Decode(b)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.CircID != c.CircID {
		t.Fatalf("CircID = %d, want %d", got.CircID, c.CircID)
	}
	if got.Command != c.Command {
		t.Fatalf("Command = %d, want %d", got.Command, c.Command)
	}
	if got.WideID {
		t.Fatalf("WideID = true, want false for 512-byte cell")
	}
	if !bytes.Equal(got.Payload[:len(payload)], payload) {
		t.Fatalf("Payload prefix mismatch")
	}
	if len(got.Payload) != 509 {
		t.Fatalf("Payload len = %d, want 509", len(got.Payload))
	}
	// rest must be zero padding
	for _, v := range got.Payload[len(payload):] {
		if v != 0 {
			t.Fatalf("non-zero padding byte")
		}
	}
}

func TestRoundtripWide514(t *testing.T) {
	payload := []byte("hello-tor-wide")
	c := Cell{CircID: 0x12345678, Command: CmdCreate2, Payload: payload, WideID: true}
	b, err := Encode(c)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(b) != V4Len {
		t.Fatalf("wide len = %d, want %d", len(b), V4Len)
	}
	got, err := Decode(b)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.CircID != c.CircID {
		t.Fatalf("CircID = %x, want %x", got.CircID, c.CircID)
	}
	if got.Command != c.Command {
		t.Fatalf("Command = %d, want %d", got.Command, c.Command)
	}
	if !got.WideID {
		t.Fatalf("WideID = false, want true for 514-byte cell")
	}
	if !bytes.Equal(got.Payload[:len(payload)], payload) {
		t.Fatalf("Payload prefix mismatch")
	}
	if len(got.Payload) != 509 {
		t.Fatalf("Payload len = %d, want 509", len(got.Payload))
	}
}

func TestRoundtripMaxPayload509(t *testing.T) {
	full := bytes.Repeat([]byte{0xAB}, 509)
	for _, wide := range []bool{false, true} {
		c := Cell{CircID: 42, Command: CmdNetinfo, Payload: full, WideID: wide}
		b, err := Encode(c)
		if err != nil {
			t.Fatalf("wide=%v Encode: %v", wide, err)
		}
		got, err := Decode(b)
		if err != nil {
			t.Fatalf("wide=%v Decode: %v", wide, err)
		}
		if !bytes.Equal(got.Payload, full) {
			t.Fatalf("wide=%v full payload mismatch", wide)
		}
	}
}

func TestPayloadOverflow(t *testing.T) {
	big := bytes.Repeat([]byte{0xFF}, 510)
	for _, wide := range []bool{false, true} {
		_, err := Encode(Cell{CircID: 1, Command: CmdRelay, Payload: big, WideID: wide})
		if err == nil {
			t.Fatalf("wide=%v expected payload overflow error", wide)
		}
	}
}

func TestCircIDOverflowNarrow(t *testing.T) {
	_, err := Encode(Cell{CircID: 0x1_0000, Command: CmdRelay, Payload: []byte("x")})
	if err == nil {
		t.Fatalf("expected circid overflow error for narrow cell")
	}
}

func TestDecodeBadLen(t *testing.T) {
	for _, n := range []int{0, 1, 511, 513, 515, 1024} {
		if _, err := Decode(make([]byte, n)); err == nil {
			t.Fatalf("len %d: expected bad cell len error", n)
		}
	}
}

func TestEmptyPayloadRoundtrip(t *testing.T) {
	c := Cell{CircID: 7, Command: CmdPadding}
	for _, wide := range []bool{false, true} {
		c.WideID = wide
		b, err := Encode(c)
		if err != nil {
			t.Fatalf("wide=%v Encode: %v", wide, err)
		}
		got, err := Decode(b)
		if err != nil {
			t.Fatalf("wide=%v Decode: %v", wide, err)
		}
		if got.CircID != c.CircID || got.Command != c.Command {
			t.Fatalf("wide=%v header mismatch", wide)
		}
	}
}
