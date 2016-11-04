package packet

import (
	"bytes"
	"io"
)

const (
	CodeConnackAccepted byte = iota
	CodeConnackRefusedProtocol
	CodeConnackRefusedIdentifier
	CodeConnackRefusedServerUnavailable
	CodeConnackRefusedUserPasswd
	CodeConnackRefusedUnauthorized
)

// The CONNACK Packet is the packet sent by the Server in response to a CONNECT Packet received
// from a Client. The first packet sent from the Server to the Client MUST be a CONNACK Packet
type ConnackPacket struct {
	AckFlags byte
	Code     byte
}

func (p *ConnackPacket) ControlType() Bit4 { return TypeCONNACK }
func (p *ConnackPacket) FixedHeader() *FixedHeader {
	return &FixedHeader{
		ControlType: TypeCONNACK,
		Flags:       0,
		Remaining:   2,
	}
}
func (p *ConnackPacket) VariableHeader() *VariableHeader {
	return &VariableHeader{
		Vary: []byte{p.AckFlags, p.Code},
	}
}
func (p *ConnackPacket) Payload() *Payload { return nil }
func (p *ConnackPacket) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	buf.Write(p.FixedHeader().Bytes())
	buf.Write(p.VariableHeader().Vary)
	return buf.WriteTo(w)
}

// ParseConnackPacketFromReader read from a reader and parse its data into a ConnackPacket.
func ParseConnackPacketFromReader(r io.Reader, first byte) (p *ConnackPacket, err error) {
	buf := make([]byte, 4)
	_, err = io.ReadFull(r, buf[1:])
	if err != nil {
		return
	}
	buf[0] = first
	return ParseConnackPacket(buf)
}

// ParseConnackPacket parse the data in bytes into a ConnackPacket.
func ParseConnackPacket(b []byte) (p *ConnackPacket, err error) {
	if len(b) < 4 {
		err = ErrLessData
		return
	}
	if byte(TypeCONNACK) != b[0]>>4 {
		err = ErrControlType
		return
	}
	if 0 != byte(b[0]&0x0f) {
		err = ErrFixedHeaderFlags
		return
	}
	if 2 != b[1] {
		err = ErrRemainLength
		return
	}
	p = &ConnackPacket{
		AckFlags: b[2],
		Code:     b[3],
	}
	return
}
