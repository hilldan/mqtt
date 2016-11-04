package packet

import "io"

// A PINGRESP Packet is sent by the Server to the Client in response to a PINGREQ Packet. It indicates
// that the Server is alive.
// This Packet is used in Keep Alive processing, see Section 3.1.2.10 for more details.
type PingrespPacket struct {
}

func (p *PingrespPacket) ControlType() Bit4 { return TypePINGRESP }
func (p *PingrespPacket) FixedHeader() *FixedHeader {
	return &FixedHeader{
		ControlType: TypePINGRESP,
		Flags:       0,
		Remaining:   0,
	}
}
func (p *PingrespPacket) VariableHeader() *VariableHeader {
	return nil
}
func (p *PingrespPacket) Payload() *Payload {
	return nil
}
func (p *PingrespPacket) WriteTo(w io.Writer) (n int64, err error) {
	m, err := w.Write(p.FixedHeader().Bytes())
	n = int64(m)
	return
}

func ParsePingrespPacketFromReader(r io.Reader, first byte) (p *PingrespPacket, err error) {
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
	p = &PingrespPacket{}
	return
}
func ParsePingrespPacket(b []byte) (p *PingrespPacket, err error) {
	if len(b) < 2 {
		err = ErrLessData
		return
	}
	if byte(TypePINGRESP) != b[0]>>4 {
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

	p = &PingrespPacket{}
	return
}
