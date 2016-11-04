package packet

import (
	"bytes"
	"io"
)

// A PUBREC Packet is the response to a PUBLISH Packet with QoS 2. It is the second packet of the QoS
// 2 protocol exchange.
type PubrecPacket struct {
	PacketId Integer
}

func (p *PubrecPacket) ControlType() Bit4 { return TypePUBREC }
func (p *PubrecPacket) FixedHeader() *FixedHeader {
	return &FixedHeader{
		ControlType: TypePUBREC,
		Flags:       0,
		Remaining:   2,
	}
}
func (p *PubrecPacket) VariableHeader() *VariableHeader {
	return &VariableHeader{
		Vary: p.PacketId.Bytes(),
	}
}
func (p *PubrecPacket) Payload() *Payload {
	return nil
}
func (p *PubrecPacket) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	buf.Write(p.FixedHeader().Bytes())
	buf.Write(p.VariableHeader().Vary)
	return buf.WriteTo(w)
}

func ParsePubrecPacketFromReader(r io.Reader, first byte) (p *PubrecPacket, err error) {
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
	p = &PubrecPacket{
		PacketId: IntegerFrom(b[1], b[2]),
	}
	return
}
func ParsePubrecPacket(b []byte) (p *PubrecPacket, err error) {
	if len(b) < 4 {
		err = ErrLessData
		return
	}
	if byte(TypePUBREC<<4) != b[0] {
		err = ErrControlType
		return
	}
	if 2 != b[1] {
		err = ErrRemainLength
		return
	}
	p = &PubrecPacket{
		PacketId: IntegerFrom(b[2], b[3]),
	}
	return
}
