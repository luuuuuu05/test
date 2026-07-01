package udp

import (
	"encoding/binary"
	"errors"
	"math"

	"mrflag/pkg/protocol"
)

type mpReader struct {
	b []byte
	i int
}

func DecodePosPacket(b []byte) (protocol.PosPacket, error) {
	r := &mpReader{b: b}
	n, err := r.readArrayLen()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	if n != 10 {
		return protocol.PosPacket{}, errors.New("position packet must be a 10-item msgpack array")
	}
	roomID, err := r.readString()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	playerID, err := r.readString()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	seq, err := r.readUint32()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	ts, err := r.readInt64()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	x, err := r.readFloat32()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	y, err := r.readFloat32()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	z, err := r.readFloat32()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	rotY, err := r.readFloat32()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	headPitch, err := r.readFloat32()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	flags, err := r.readUint8()
	if err != nil {
		return protocol.PosPacket{}, err
	}
	return protocol.PosPacket{
		RoomID:    roomID,
		PlayerID:  playerID,
		Seq:       seq,
		Ts:        ts,
		X:         x,
		Y:         y,
		Z:         z,
		RotY:      rotY,
		HeadPitch: headPitch,
		Flags:     flags,
	}, nil
}

func EncodeRelayPacket(pkt protocol.RelayPacket) []byte {
	out := []byte{0x99}
	out = appendString(out, pkt.FromPlayerID)
	out = appendUint32(out, pkt.Seq)
	out = appendInt64(out, pkt.ServerTs)
	out = appendFloat32(out, pkt.X)
	out = appendFloat32(out, pkt.Y)
	out = appendFloat32(out, pkt.Z)
	out = appendFloat32(out, pkt.RotY)
	out = appendFloat32(out, pkt.HeadPitch)
	out = appendUint8(out, pkt.Flags)
	return out
}

func (r *mpReader) readByte() (byte, error) {
	if r.i >= len(r.b) {
		return 0, errors.New("unexpected end of msgpack")
	}
	v := r.b[r.i]
	r.i++
	return v, nil
}

func (r *mpReader) readArrayLen() (int, error) {
	prefix, err := r.readByte()
	if err != nil {
		return 0, err
	}
	if prefix >= 0x90 && prefix <= 0x9F {
		return int(prefix & 0x0F), nil
	}
	if prefix == 0xDC {
		b, err := r.readN(2)
		if err != nil {
			return 0, err
		}
		return int(binary.BigEndian.Uint16(b)), nil
	}
	return 0, errors.New("expected msgpack array")
}

func (r *mpReader) readString() (string, error) {
	prefix, err := r.readByte()
	if err != nil {
		return "", err
	}
	var n int
	switch {
	case prefix >= 0xA0 && prefix <= 0xBF:
		n = int(prefix & 0x1F)
	case prefix == 0xD9:
		b, err := r.readByte()
		if err != nil {
			return "", err
		}
		n = int(b)
	case prefix == 0xDA:
		b, err := r.readN(2)
		if err != nil {
			return "", err
		}
		n = int(binary.BigEndian.Uint16(b))
	case prefix == 0xC4:
		b, err := r.readByte()
		if err != nil {
			return "", err
		}
		n = int(b)
	case prefix == 0xC5:
		b, err := r.readN(2)
		if err != nil {
			return "", err
		}
		n = int(binary.BigEndian.Uint16(b))
	default:
		return "", errors.New("expected msgpack string")
	}
	b, err := r.readN(n)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *mpReader) readUint8() (uint8, error) {
	prefix, err := r.readByte()
	if err != nil {
		return 0, err
	}
	switch {
	case prefix <= 0x7F:
		return prefix, nil
	case prefix == 0xCC:
		b, err := r.readByte()
		return b, err
	case prefix == 0xCD:
		b, err := r.readN(2)
		if err != nil {
			return 0, err
		}
		return uint8(binary.BigEndian.Uint16(b)), nil
	default:
		return 0, errors.New("expected msgpack uint8")
	}
}

func (r *mpReader) readUint32() (uint32, error) {
	prefix, err := r.readByte()
	if err != nil {
		return 0, err
	}
	switch {
	case prefix <= 0x7F:
		return uint32(prefix), nil
	case prefix == 0xCC:
		b, err := r.readByte()
		return uint32(b), err
	case prefix == 0xCD:
		b, err := r.readN(2)
		if err != nil {
			return 0, err
		}
		return uint32(binary.BigEndian.Uint16(b)), nil
	case prefix == 0xCE:
		b, err := r.readN(4)
		if err != nil {
			return 0, err
		}
		return binary.BigEndian.Uint32(b), nil
	case prefix == 0xD2:
		b, err := r.readN(4)
		if err != nil {
			return 0, err
		}
		return uint32(int32(binary.BigEndian.Uint32(b))), nil
	default:
		return 0, errors.New("expected msgpack uint32")
	}
}

func (r *mpReader) readInt64() (int64, error) {
	prefix, err := r.readByte()
	if err != nil {
		return 0, err
	}
	switch {
	case prefix <= 0x7F:
		return int64(prefix), nil
	case prefix >= 0xE0:
		return int64(int8(prefix)), nil
	case prefix == 0xCC:
		b, err := r.readByte()
		return int64(b), err
	case prefix == 0xCD:
		b, err := r.readN(2)
		if err != nil {
			return 0, err
		}
		return int64(binary.BigEndian.Uint16(b)), nil
	case prefix == 0xCE:
		b, err := r.readN(4)
		if err != nil {
			return 0, err
		}
		return int64(binary.BigEndian.Uint32(b)), nil
	case prefix == 0xCF:
		b, err := r.readN(8)
		if err != nil {
			return 0, err
		}
		return int64(binary.BigEndian.Uint64(b)), nil
	case prefix == 0xD0:
		b, err := r.readByte()
		return int64(int8(b)), err
	case prefix == 0xD1:
		b, err := r.readN(2)
		if err != nil {
			return 0, err
		}
		return int64(int16(binary.BigEndian.Uint16(b))), nil
	case prefix == 0xD2:
		b, err := r.readN(4)
		if err != nil {
			return 0, err
		}
		return int64(int32(binary.BigEndian.Uint32(b))), nil
	case prefix == 0xD3:
		b, err := r.readN(8)
		if err != nil {
			return 0, err
		}
		return int64(binary.BigEndian.Uint64(b)), nil
	default:
		return 0, errors.New("expected msgpack int64")
	}
}

func (r *mpReader) readFloat32() (float32, error) {
	prefix, err := r.readByte()
	if err != nil {
		return 0, err
	}
	switch prefix {
	case 0xCA:
		b, err := r.readN(4)
		if err != nil {
			return 0, err
		}
		return math.Float32frombits(binary.BigEndian.Uint32(b)), nil
	case 0xCB:
		b, err := r.readN(8)
		if err != nil {
			return 0, err
		}
		return float32(math.Float64frombits(binary.BigEndian.Uint64(b))), nil
	default:
		if prefix <= 0x7F {
			return float32(prefix), nil
		}
		return 0, errors.New("expected msgpack float")
	}
}

func (r *mpReader) readN(n int) ([]byte, error) {
	if n < 0 || r.i+n > len(r.b) {
		return nil, errors.New("unexpected end of msgpack")
	}
	v := r.b[r.i : r.i+n]
	r.i += n
	return v, nil
}

func appendString(out []byte, s string) []byte {
	n := len(s)
	switch {
	case n <= 31:
		out = append(out, 0xA0|byte(n))
	case n <= 0xFF:
		out = append(out, 0xD9, byte(n))
	case n <= 0xFFFF:
		out = append(out, 0xDA, byte(n>>8), byte(n))
	default:
		out = append(out, 0xDB, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	return append(out, s...)
}

func appendUint8(out []byte, v uint8) []byte {
	if v <= 0x7F {
		return append(out, v)
	}
	return append(out, 0xCC, v)
}

func appendUint32(out []byte, v uint32) []byte {
	switch {
	case v <= 0x7F:
		return append(out, byte(v))
	case v <= 0xFF:
		return append(out, 0xCC, byte(v))
	case v <= 0xFFFF:
		return append(out, 0xCD, byte(v>>8), byte(v))
	default:
		return append(out, 0xCE, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	}
}

func appendInt64(out []byte, v int64) []byte {
	if v >= 0 && v <= 0x7F {
		return append(out, byte(v))
	}
	out = append(out, 0xD3)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(v))
	return append(out, buf[:]...)
}

func appendFloat32(out []byte, v float32) []byte {
	out = append(out, 0xCA)
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], math.Float32bits(v))
	return append(out, buf[:]...)
}
