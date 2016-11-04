package packet

import (
	"bytes"
	"io"
)

// The SUBSCRIBE Packet is sent from the Client to the Server to create one or more Subscriptions. Each
// Subscription registers a Client’s interest in one or more Topics. The Server sends PUBLISH Packets to
// the Client in order to forward Application Messages that were published to Topics that match these
// Subscriptions. The SUBSCRIBE Packet also specifies (for each Subscription) the maximum QoS with
// which the Server can send Application Messages to the Client.
type SubscribePacket struct {
	//fixed header
	remainl int
	//variable header
	PacketId Integer
	//payload
	TopicFilters []TopicFilter
}

// The payload of a SUBSCRIBE Packet contains a list of Topic Filters indicating the Topics to which the
// Client wants to subscribe. The Topic Filters in a SUBSCRIBE packet payload MUST be UTF-8 encoded
// strings as defined in Section 1.5.3 [MQTT-3.8.3-1]. A Server SHOULD support Topic filters that contain
// the wildcard characters defined in Section 4.7.1. If it chooses not to support topic filters that contain
// wildcard characters it MUST reject any Subscription request whose filter contains them [MQTT-3.8.3-2].
// Each filter is followed by a byte called the Requested QoS. This gives the maximum QoS level at which
// the Server can send Application Messages to the Client.
// The payload of a SUBSCRIBE packet MUST contain at least one Topic Filter / QoS pair. A SUBSCRIBE
// packet with no payload is a protocol violation [MQTT-3.8.3-3]. See section 4.8 for information about
// handling errors.
type TopicFilter struct {
	Topic String
	Qos   Bit2
}

func (p *SubscribePacket) ControlType() Bit4 { return TypeSUBSCRIBE }
func (p *SubscribePacket) FixedHeader() *FixedHeader {
	if p.remainl == 0 {
		p.Payload()
	}
	return &FixedHeader{
		ControlType: TypeSUBSCRIBE,
		Flags:       2,
		Remaining:   p.remainl,
	}
}
func (p *SubscribePacket) VariableHeader() *VariableHeader {
	return &VariableHeader{
		Vary: p.PacketId.Bytes(),
	}
}
func (p *SubscribePacket) Payload() *Payload {
	if p.TopicFilters == nil {
		return nil
	}
	var buf bytes.Buffer
	for _, v := range p.TopicFilters {
		buf.Write(v.Topic.Bytes())
		buf.WriteByte(byte(v.Qos))
	}
	b := buf.Bytes()
	p.remainl = 2 + len(b)
	return &Payload{
		Content: b,
	}
}
func (p *SubscribePacket) WriteTo(w io.Writer) (n int64, err error) {
	pay := p.Payload().Content
	var buf bytes.Buffer
	buf.Write(p.FixedHeader().Bytes())
	buf.Write(p.VariableHeader().Vary)
	buf.Write(pay)
	return buf.WriteTo(w)
}

func ParseSubscribePacketFromReader(r io.Reader, first byte) (p *SubscribePacket, err error) {
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
	//parse variable header
	packetId := IntegerFrom(rc[0], rc[1])

	//parse payload
	start := 2
	filter := make([]TopicFilter, 0)
	for start < rl {
		k := StringFrom(rc[start:])
		start += len(k) + 2
		v := Bit2(rc[start])
		start++
		filter = append(filter, TopicFilter{Topic: k, Qos: v})
	}

	p = &SubscribePacket{
		remainl:      rl,
		PacketId:     packetId,
		TopicFilters: filter,
	}
	return

}
func ParseSubscribePacket(b []byte) (p *SubscribePacket, err error) {
	if len(b) < 8 {
		err = ErrLessData
		return
	}

	//parse fixed header
	if byte(TypeSUBSCRIBE) != b[0]>>4 {
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

	//parse variable header
	packetId := IntegerFrom(rc[0], rc[1])

	//parse payload
	start := 2
	filter := make([]TopicFilter, 0)
	for start < rl {
		k := StringFrom(rc[start:])
		start += len(k) + 2
		v := Bit2(rc[start])
		start++
		filter = append(filter, TopicFilter{Topic: k, Qos: v})
	}

	p = &SubscribePacket{
		remainl:      rl,
		PacketId:     packetId,
		TopicFilters: filter,
	}
	return
}
