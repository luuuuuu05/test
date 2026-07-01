package udp

import (
	"testing"

	"mrflag/pkg/protocol"
)

func TestDecodePosPacketAndEncodeRelayPacket(t *testing.T) {
	raw := []byte{0x9A}
	raw = appendString(raw, "ROOM_ABC123")
	raw = appendString(raw, "P_001")
	raw = appendUint32(raw, 7)
	raw = appendInt64(raw, 1700000000000)
	raw = appendFloat32(raw, 1.25)
	raw = appendFloat32(raw, 0.5)
	raw = appendFloat32(raw, -2.75)
	raw = appendFloat32(raw, 90)
	raw = appendFloat32(raw, 10)
	raw = appendUint8(raw, 3)

	pkt, err := DecodePosPacket(raw)
	if err != nil {
		t.Fatalf("DecodePosPacket: %v", err)
	}
	if pkt.RoomID != "ROOM_ABC123" || pkt.PlayerID != "P_001" || pkt.Seq != 7 || pkt.Flags != 3 {
		t.Fatalf("decoded packet mismatch: %#v", pkt)
	}

	relay := EncodeRelayPacket(protocol.RelayPacket{
		FromPlayerID: pkt.PlayerID,
		Seq:          pkt.Seq,
		ServerTs:     1700000000100,
		X:            pkt.X,
		Y:            pkt.Y,
		Z:            pkt.Z,
		RotY:         pkt.RotY,
		HeadPitch:    pkt.HeadPitch,
		Flags:        pkt.Flags,
	})
	if len(relay) == 0 || relay[0] != 0x99 {
		t.Fatalf("relay prefix = %#x, want fixarray(9)", relay[0])
	}
}
