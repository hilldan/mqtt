package packet

import (
	"bytes"
	"io"
)

// A PUBACK Packet is the response to a PUBLISH Packet with QoS level 1.
type PubackPacket struct {
	PacketId Integer
}

func (p *PubackPacket) ControlType() Bit4 { return TypePUBACK }
func (p *PubackPacket) FixedHeader() *FixedHeader {
	return &FixedHeader{
		ControlType: TypePUBACK,
		Flags:       0,
		Remaining:   2,
	}
}
func (p *PubackPacket) VariableHeader() *VariableHeader {
	return &VariableHeader{
		Vary: p.PacketId.Bytes(),
	}
}
func (p *PubackPacket) Payload() *Payload {
	return nil
}
func (p *PubackPacket) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	buf.Write(p.FixedHeader().Bytes())
	buf.Write(p.VariableHeader().Vary)
	return buf.WriteTo(w)
}

func ParsePubackPacketFromReader(r io.Reader, first byte) (p *PubackPacket, err error) {
	if 0 != first&0x0f {
		err = ErrFixedHeaderFlags
		return
	}
	b := make([]byte, 3)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return
	}
	if b[0] != 2 {
		err = ErrRemainLength
		return
	}
	p = &PubackPacket{
		PacketId: IntegerFrom(b[1], b[2]),
	}
	return
}
func ParsePubackPacket(b []byte) (p *PubackPacket, err error) {
	if len(b) < 4 {
		err = ErrLessData
		return
	}
	if byte(TypePUBACK<<4) != b[0] {
		err = ErrControlType
		return
	}
	if 2 != b[1] {
		err = ErrRemainLength
		return
	}
	p = &PubackPacket{
		PacketId: IntegerFrom(b[2], b[3]),
	}
	return
}
