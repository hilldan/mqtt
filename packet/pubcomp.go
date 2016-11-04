package packet

import (
	"bytes"
	"io"
)

// The PUBCOMP Packet is the response to a PUBREL Packet. It is the fourth and final packet of the QoS
// 2 protocol exchange.
type PubcompPacket struct {
	PacketId Integer
}

func (p *PubcompPacket) ControlType() Bit4 { return TypePUBCOMP }
func (p *PubcompPacket) FixedHeader() *FixedHeader {
	return &FixedHeader{
		ControlType: TypePUBCOMP,
		Flags:       0,
		Remaining:   2,
	}
}
func (p *PubcompPacket) VariableHeader() *VariableHeader {
	return &VariableHeader{
		Vary: p.PacketId.Bytes(),
	}
}
func (p *PubcompPacket) Payload() *Payload {
	return nil
}
func (p *PubcompPacket) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	buf.Write(p.FixedHeader().Bytes())
	buf.Write(p.VariableHeader().Vary)
	return buf.WriteTo(w)
}

func ParsePubcompPacketFromReader(r io.Reader, first byte) (p *PubcompPacket, err error) {
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
	p = &PubcompPacket{
		PacketId: IntegerFrom(b[1], b[2]),
	}
	return
}
func ParsePubcompPacket(b []byte) (p *PubcompPacket, err error) {
	if len(b) < 4 {
		err = ErrLessData
		return
	}
	if byte(TypePUBCOMP<<4) != b[0] {
		err = ErrControlType
		return
	}
	if 2 != b[1] {
		err = ErrRemainLength
		return
	}
	p = &PubcompPacket{
		PacketId: IntegerFrom(b[2], b[3]),
	}
	return
}
