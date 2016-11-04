package packet

import (
	"bytes"
	"io"
)

// The UNSUBACK Packet is sent by the Server to the Client to confirm receipt of an UNSUBSCRIBE
// Packet.
type UnsubackPacket struct {
	PacketId Integer
}

func (p *UnsubackPacket) ControlType() Bit4 { return TypeUNSUBACK }
func (p *UnsubackPacket) FixedHeader() *FixedHeader {
	return &FixedHeader{
		ControlType: TypeUNSUBACK,
		Flags:       0,
		Remaining:   2,
	}
}
func (p *UnsubackPacket) VariableHeader() *VariableHeader {
	return &VariableHeader{
		Vary: p.PacketId.Bytes(),
	}
}
func (p *UnsubackPacket) Payload() *Payload {
	return nil
}
func (p *UnsubackPacket) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	buf.Write(p.FixedHeader().Bytes())
	buf.Write(p.VariableHeader().Vary)
	return buf.WriteTo(w)
}

func ParseUnsubackPacketFromReader(r io.Reader, first byte) (p *UnsubackPacket, err error) {
	if 0 != first&0x0f {
		err = ErrFixedHeaderFlags
		return
	}
	b := make([]byte, 3)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return
	}
	if 2 != b[0] {
		err = ErrRemainLength
		return
	}
	p = &UnsubackPacket{
		PacketId: IntegerFrom(b[1], b[2]),
	}
	return
}
func ParseUnsubackPacket(b []byte) (p *UnsubackPacket, err error) {
	if len(b) < 4 {
		err = ErrLessData
		return
	}
	if byte(TypeUNSUBACK) != b[0]>>4 {
		err = ErrControlType
		return
	}
	if 0 != b[0]&0x0f {
		err = ErrFixedHeaderFlags
		return
	}

	p = &UnsubackPacket{
		PacketId: IntegerFrom(b[2], b[3]),
	}
	return
}
