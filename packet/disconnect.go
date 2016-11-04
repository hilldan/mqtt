package packet

import "io"

// The DISCONNECT Packet is the final Control Packet sent from the Client to the Server. It indicates that
// the Client is disconnecting cleanly.
type DisconnectPacket struct {
}

func (p *DisconnectPacket) ControlType() Bit4 { return TypeDISCONNECT }
func (p *DisconnectPacket) FixedHeader() *FixedHeader {
	return &FixedHeader{
		ControlType: TypeDISCONNECT,
		Flags:       0,
		Remaining:   0,
	}
}
func (p *DisconnectPacket) VariableHeader() *VariableHeader {
	return nil
}
func (p *DisconnectPacket) Payload() *Payload {
	return nil
}
func (p *DisconnectPacket) WriteTo(w io.Writer) (n int64, err error) {
	m, err := w.Write(p.FixedHeader().Bytes())
	n = int64(m)
	return
}

func ParseDisconnectPacketFromReader(r io.Reader, first byte) (p *DisconnectPacket, err error) {
	if 0 != first&0x0f {
		err = ErrFixedHeaderFlags
		return
	}
	var b [1]byte
	_, err = r.Read(b[:])
	if err != nil {
		return
	}
	if 0 != b[0] {
		err = ErrRemainLength
		return
	}
	p = &DisconnectPacket{}
	return
}
func ParseDisconnectPacket(b []byte) (p *DisconnectPacket, err error) {
	if len(b) < 2 {
		err = ErrLessData
		return
	}
	if byte(TypeDISCONNECT) != b[0]>>4 {
		err = ErrControlType
		return
	}
	if 0 != b[0]&0x0f {
		err = ErrFixedHeaderFlags
		return
	}
	if 0 != b[1] {
		err = ErrRemainLength
		return
	}

	p = &DisconnectPacket{}
	return
}
