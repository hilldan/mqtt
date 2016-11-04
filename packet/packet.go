package packet

import "io"

const (
	TypeCONNECT Bit4 = iota + 1
	TypeCONNACK
	TypePUBLISH
	TypePUBACK
	TypePUBREC
	TypePUBREL
	TypePUBCOMP
	TypeSUBSCRIBE
	TypeSUBACK
	TypeUNSUBSCRIBE
	TypeUNSUBACK
	TypePINGREQ
	TypePINGRESP
	TypeDISCONNECT
)

const (
	QoS0 Bit2 = iota
	QoS1
	QoS2
)

// MQTT Control Packet:
// A packet of information that is sent across the Network Connection. The MQTT specification defines
// fourteen different types of Control Packet, one of which (the PUBLISH packet) is used to convey
// Application Messages.
// The MQTT protocol works by exchanging a series of MQTT Control Packets in a defined way.
type ControlPacketer interface {
	ControlType() Bit4
	FixedHeader() *FixedHeader
	VariableHeader() *VariableHeader
	Payload() *Payload
	io.WriterTo
}

// Fixed header, present in all MQTT Control Packets
type FixedHeader struct {
	ControlType Bit4 // bits [7-4] in byte 1
	Flags       byte // bits [3-0] in byte 1
	Remaining   int  // <=268435455 (0xFFFFFF7F), 1-4 bytes
}

func (fh *FixedHeader) Bytes() []byte {
	b := []byte{byte(fh.ControlType<<4) | fh.Flags&0x0f}
	return append(b, encodeRemainLength(fh.Remaining)...)
}

// Some types of MQTT Control Packets contain a variable header component. It resides between the fixed
// header and the payload. The content of the variable header varies depending on the Packet type. The
// Packet Identifier field of variable header is common in several packet types.
type VariableHeader struct {
	Vary []byte
}

// Some MQTT Control Packets contain a payload as the final part of the packet, as described in Chapter 3.
// In the case of the PUBLISH packet this is the Application Message. Table 2.6 - Control Packets that
// contain a Payload lists the Control Packets that require a Payload.
type Payload struct {
	Content []byte
}

//
func ParsePacket(r io.Reader) (p ControlPacketer, err error) {
	var b [1]byte
	_, err = r.Read(b[:])
	if err != nil {
		return
	}
	switch Bit4(b[0] >> 4) {
	case TypeCONNECT:
		p, err = ParseConnectPacketFromReader(r, b[0])
	case TypeCONNACK:
		p, err = ParseConnackPacketFromReader(r, b[0])
	case TypePUBLISH:
		p, err = ParsePublishPacketFromReader(r, b[0])
	case TypePUBACK:
		p, err = ParsePubackPacketFromReader(r, b[0])
	case TypePUBREC:
		p, err = ParsePubrecPacketFromReader(r, b[0])
	case TypePUBREL:
		p, err = ParsePubrelPacketFromReader(r, b[0])
	case TypePUBCOMP:
		p, err = ParsePubcompPacketFromReader(r, b[0])
	case TypeSUBSCRIBE:
		p, err = ParseSubscribePacketFromReader(r, b[0])
	case TypeSUBACK:
		p, err = ParseSubackPacketFromReader(r, b[0])
	case TypeUNSUBSCRIBE:
		p, err = ParseUnsubscribePacketFromReader(r, b[0])
	case TypeUNSUBACK:
		p, err = ParseUnsubackPacketFromReader(r, b[0])
	case TypePINGREQ:
		p, err = ParsePingreqPacketFromReader(r, b[0])
	case TypePINGRESP:
		p, err = ParsePingrespPacketFromReader(r, b[0])
	case TypeDISCONNECT:
		p, err = ParseDisconnectPacketFromReader(r, b[0])
	default:
		err = ErrControlType
	}
	return
}

const (
	t3 = 128 * 128 * 128
	t2 = 128 * 128
	t1 = 128 //0x80
)

// The Remaining Length is encoded using a variable length encoding scheme which uses a single byte for
// values up to 127. Larger values are handled as follows. The least significant seven bits of each byte
// encode the data, and the most significant bit is used to indicate that there are following bytes in the
// representation. Thus each byte encodes 128 values and a "continuation bit". The maximum number of
// bytes in the Remaining Length field is four.
// note the input length ought to be less than 268435455 (0xFFFFFF7F) and >=0.
func encodeRemainLength(l int) []byte {
	r := []byte{byte(l % t1)}
	n := l / t1
	if n == 0 {
		return r
	}

	r[0] |= 0x80
	r = append(r, byte(n))
	p := l / t2
	if p == 0 {
		return r
	}

	r[1] |= 0x80
	r = append(r, byte(p))
	q := l / t3
	if q == 0 {
		return r
	}

	r[2] |= 0x80
	r = append(r, byte(q))
	return r
}

// decodeRemainLength return the remainning content and its length
func decodeRemainLength(b []byte) (remainl int, remainContent []byte) {
	i := 0
	for ; i < len(b) && i < 4; i++ {
		t := 1
		switch i {
		case 1:
			t = t1
		case 2:
			t = t2
		case 3:
			t = t3
		}
		if b[i]&0x80 == 0 {
			remainl += int(b[i]) * t
			i++ //step once
			break
		}
		remainl += int(b[i]&0x7f) * t
	}
	remainContent = b[i:]
	return
}

func remainSize(l int) int {
	return len(encodeRemainLength(l))
}

//
func decodeRemainLengthFromReader(r io.Reader) (remainl int, err error) {
	var b [1]byte
	i := 0
	for ; i < 4; i++ {
		_, err = r.Read(b[:])
		if err != nil {
			break
		}
		t := 1
		switch i {
		case 1:
			t = t1
		case 2:
			t = t2
		case 3:
			t = t3
		}
		if b[0]&0x80 == 0 {
			remainl += int(b[0]) * t
			break
		}
		remainl += int(b[0]&0x7f) * t
	}
	return
}
