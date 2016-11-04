package packet

import (
	"bytes"
	"io"
)

// An UNSUBSCRIBE Packet is sent by the Client to the Server, to unsubscribe from topics.
type UnsubscribePacket struct {
	remainl     int
	PacketId    Integer
	TopicFilter []String
}

func (p *UnsubscribePacket) ControlType() Bit4 { return TypeUNSUBSCRIBE }
func (p *UnsubscribePacket) FixedHeader() *FixedHeader {
	if p.remainl == 0 {
		p.Payload()
	}
	return &FixedHeader{
		ControlType: TypeUNSUBSCRIBE,
		Flags:       2,
		Remaining:   p.remainl,
	}
}
func (p *UnsubscribePacket) VariableHeader() *VariableHeader {
	return &VariableHeader{
		Vary: p.PacketId.Bytes(),
	}
}
func (p *UnsubscribePacket) Payload() *Payload {
	b := []byte{}
	for _, v := range p.TopicFilter {
		b = append(b, v.Bytes()...)
	}
	p.remainl = 2 + len(b)
	return &Payload{
		Content: b,
	}
}
func (p *UnsubscribePacket) WriteTo(w io.Writer) (n int64, err error) {
	pay := p.Payload().Content
	var buf bytes.Buffer
	buf.Write(p.FixedHeader().Bytes())
	buf.Write(p.VariableHeader().Vary)
	buf.Write(pay)
	return buf.WriteTo(w)
}

func ParseUnsubscribePacketFromReader(r io.Reader, first byte) (p *UnsubscribePacket, err error) {
	if 2 != first&0x0f {
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

	var topicFilter []String
	i := 2
	for i < rl {
		tf := StringFrom(rc[i:])
		topicFilter = append(topicFilter, tf)
		i += 2 + len(tf)
	}

	p = &UnsubscribePacket{
		remainl:     rl,
		PacketId:    packetId,
		TopicFilter: topicFilter,
	}
	return

}
func ParseUnsubscribePacket(b []byte) (p *UnsubscribePacket, err error) {
	if len(b) < 7 {
		err = ErrLessData
		return
	}
	if byte(TypeUNSUBSCRIBE) != b[0]>>4 {
		err = ErrControlType
		return
	}
	if 2 != b[0]&0x0f {
		err = ErrFixedHeaderFlags
		return
	}
	rl, rc := decodeRemainLength(b[1:])
	if len(rc) < rl {
		err = ErrRemainLength
		return
	}
	packetId := IntegerFrom(rc[0], rc[1])

	var topicFilter []String
	i := 2
	for i < rl {
		tf := StringFrom(rc[i:])
		topicFilter = append(topicFilter, tf)
		i += 2 + len(tf)
	}

	p = &UnsubscribePacket{
		remainl:     rl,
		PacketId:    packetId,
		TopicFilter: topicFilter,
	}
	return
}
