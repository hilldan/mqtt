package packet

import (
	"bytes"
	"io"
)

// A PUBREL Packet is the response to a PUBREC Packet. It is the third packet of the QoS 2 protocol
// exchange.
type PubrelPacket struct {
	PacketId Integer
}

func (p *PubrelPacket) ControlType() Bit4 { return TypePUBREL }
func (p *PubrelPacket) FixedHeader() *FixedHeader {
	return &FixedHeader{
		ControlType: TypePUBREL,
		Flags:       2,
		Remaining:   2,
	}
}
func (p *PubrelPacket) VariableHeader() *VariableHeader {
	return &VariableHeader{
		Vary: p.PacketId.Bytes(),
	}
}
func (p *PubrelPacket) Payload() *Payload {
	return nil
}
func (p *PubrelPacket) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	buf.Write(p.FixedHeader().Bytes())
	buf.Write(p.VariableHeader().Vary)
	return buf.WriteTo(w)
}

func ParsePubrelPacketFromReader(r io.Reader, first byte) (p *PubrelPacket, err error) {
	if 2 != first&0x0f {
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
	p = &PubrelPacket{
		PacketId: IntegerFrom(b[1], b[2]),
	}
	return
}
func ParsePubrelPacket(b []byte) (p *PubrelPacket, err error) {
	if len(b) < 4 {
		err = ErrLessData
		return
	}
	if byte(TypePUBREL) != b[0]>>4 {
		err = ErrControlType
		return
	}
	if b[0]&0x02 != 2 {
		err = ErrFixedHeaderFlags
		return
	}
	if 2 != b[1] {
		err = ErrRemainLength
		return
	}
	p = &PubrelPacket{
		PacketId: IntegerFrom(b[2], b[3]),
	}
	return
}
