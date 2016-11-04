package packet

import (
	"bytes"
	"io"
)

// A PUBLISH Control Packet is sent from a Client to a Server or from Server to a Client to transport an
// Application Message.
type PublishPacket struct {
	//fixed header
	Dup          Bool
	Qos          Bit2
	Retain       Bool
	remainlength int

	//variable header
	TopicName String
	PacketId  Integer //The Packet Identifier field is only present in PUBLISH Packets where the QoS level is 1 or 2

	//payload
	ApplicationMessage String
	payloadLength      int
}

func (p *PublishPacket) ControlType() Bit4 { return TypePUBLISH }
func (p *PublishPacket) FixedHeader() *FixedHeader {
	if p.remainlength == 0 {
		p.VariableHeader()
	}
	return &FixedHeader{
		ControlType: TypePUBLISH,
		Flags:       p.Dup.Byte()<<3 | byte(p.Qos<<1) | p.Retain.Byte(),
		Remaining:   p.remainlength,
	}
}
func (p *PublishPacket) VariableHeader() *VariableHeader {
	b := p.TopicName.Bytes()
	if p.Qos != 0 {
		b = append(b, p.PacketId.Bytes()...)
	}
	if p.payloadLength == 0 {
		p.Payload()
	}
	p.remainlength = len(b) + p.payloadLength
	return &VariableHeader{
		Vary: b,
	}
}
func (p *PublishPacket) Payload() *Payload {
	b := p.ApplicationMessage.Bytes()
	p.payloadLength = len(b)
	return &Payload{
		Content: b,
	}
}
func (p *PublishPacket) WriteTo(w io.Writer) (n int64, err error) {
	payload := p.Payload().Content
	vary := p.VariableHeader().Vary
	fixed := p.FixedHeader().Bytes()
	var buf bytes.Buffer
	buf.Write(fixed)
	buf.Write(vary)
	buf.Write(payload)
	return buf.WriteTo(w)
}

func ParsePublishPacketFromReader(r io.Reader, first byte) (p *PublishPacket, err error) {
	dup := Bool(first&0x08>>3 == 1)
	qos := Bit2(first & 0x06 >> 1)
	retain := Bool(first&0x01 == 1)

	if qos == QoS0 {
		dup = false
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
	//parse variable header
	topicName := StringFrom(rc)
	start := len(topicName) + 2
	var packetId Integer
	if qos != 0 {
		packetId = IntegerFrom(rc[start], rc[start+1])
		start += 2
	}

	//parse payload
	pl := rl - start
	if pl < 0 {
		err = ErrLessData
		return
	}
	message := StringFrom(rc[start:])

	p = &PublishPacket{
		Dup:                dup,
		Qos:                qos,
		Retain:             retain,
		remainlength:       rl,
		TopicName:          topicName,
		PacketId:           packetId,
		ApplicationMessage: message,
		payloadLength:      pl,
	}
	return
}

func ParsePublishPacket(b []byte) (p *PublishPacket, err error) {
	if len(b) < 5 {
		err = ErrLessData
		return
	}

	//parse fixed header
	if byte(TypePUBLISH) != b[0]>>4 {
		err = ErrControlType
		return
	}
	dup := Bool(b[0]&0x08>>3 == 1)
	qos := Bit2(b[0] & 0x06 >> 1)
	retain := Bool(b[0]&0x01 == 1)

	if qos == QoS0 {
		dup = false
	}
	rl, rc := decodeRemainLength(b[1:])
	if rl < len(rc) {
		err = ErrRemainLength
		return
	}

	//parse variable header
	topicName := StringFrom(rc)
	start := len(topicName) + 2
	var packetId Integer
	if qos != 0 {
		packetId = IntegerFrom(rc[start], rc[start+1])
		start += 2
	}

	//parse payload
	pl := rl - start
	if pl < 0 {
		err = ErrLessData
		return
	}
	message := StringFrom(rc[start:])

	p = &PublishPacket{
		Dup:                dup,
		Qos:                qos,
		Retain:             retain,
		remainlength:       rl,
		TopicName:          topicName,
		PacketId:           packetId,
		ApplicationMessage: message,
		payloadLength:      pl,
	}
	return
}
