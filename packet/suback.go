package packet

import (
	"bytes"
	"io"
)

const (
	CodeSubackSuccess0 byte = iota
	CodeSubackSuccess1
	CodeSubackSuccess2

	CodeSubackFailure byte = 0x80
)

// A SUBACK Packet is sent by the Server to the Client to confirm receipt
// and processing of a SUBSCRIBE Packet.
// A SUBACK Packet contains a list of return codes, that specify the maximum QoS level
// that was granted in each Subscription that was requested by the SUBSCRIBE.
type SubackPacket struct {
	//fixed header
	remainl int
	//variable header
	PacketId Integer
	//payload
	// The payload contains a list of return codes.
	// Each return code corresponds to a Topic Filter in the
	// SUBSCRIBE Packet being acknowledged. The order of return codes in the SUBACK Packet MUST
	// match the order of Topic Filters in the SUBSCRIBE Packet
	Code []byte
}

func (p *SubackPacket) ControlType() Bit4 { return TypeSUBACK }
func (p *SubackPacket) FixedHeader() *FixedHeader {
	if p.remainl == 0 {
		p.Payload()
	}
	return &FixedHeader{
		ControlType: TypeSUBACK,
		Flags:       0,
		Remaining:   p.remainl,
	}
}
func (p *SubackPacket) VariableHeader() *VariableHeader {
	return &VariableHeader{
		Vary: p.PacketId.Bytes(),
	}
}
func (p *SubackPacket) Payload() *Payload {
	p.remainl = 2 + len(p.Code)
	return &Payload{
		Content: p.Code,
	}
}
func (p *SubackPacket) WriteTo(w io.Writer) (n int64, err error) {
	pay := p.Payload().Content
	var buf bytes.Buffer
	buf.Write(p.FixedHeader().Bytes())
	buf.Write(p.VariableHeader().Vary)
	buf.Write(pay)
	return buf.WriteTo(w)
}

func ParseSubackPacketFromReader(r io.Reader, first byte) (p *SubackPacket, err error) {
	if 0 != first&0x0f {
		err = ErrFixedHeaderFlags
		return
	}
	rl, err := decodeRemainLengthFromReader(r)
	if err != nil {
		return
	}
	if rl == 0 {
		err = ErrRemainLength
		return
	}
	rc := make([]byte, rl)
	_, err = io.ReadFull(r, rc)
	if err != nil {
		return
	}
	packetId := IntegerFrom(rc[0], rc[1])

	p = &SubackPacket{
		remainl:  rl,
		PacketId: packetId,
		Code:     rc[2:],
	}
	return
}
func ParseSubackPacket(b []byte) (p *SubackPacket, err error) {
	if len(b) < 5 {
		err = ErrLessData
		return
	}
	if byte(TypeSUBACK<<4) != b[0] {
		err = ErrControlType
		return
	}
	rl, rc := decodeRemainLength(b[1:])
	if len(rc) < rl {
		err = ErrRemainLength
		return
	}
	packetId := IntegerFrom(rc[0], rc[1])

	p = &SubackPacket{
		remainl:  rl,
		PacketId: packetId,
		Code:     rc[2:],
	}
	return
}
