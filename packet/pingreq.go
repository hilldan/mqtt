package packet

import "io"

// The PINGREQ Packet is sent from a Client to the Server. It can be used to:
//  1. Indicate to the Server that the Client is alive in the absence of any other Control Packets being
// sent from the Client to the Server.
//  2. Request that the Server responds to confirm that it is alive.
//  3. Exercise the network to indicate that the Network Connection is active.
// This Packet is used in Keep Alive processing, see Section 3.1.2.10 for more details.
type PingreqPacket struct {
}

func (p *PingreqPacket) ControlType() Bit4 { return TypePINGREQ }
func (p *PingreqPacket) FixedHeader() *FixedHeader {
	return &FixedHeader{
		ControlType: TypePINGREQ,
		Flags:       0,
		Remaining:   0,
	}
}
func (p *PingreqPacket) VariableHeader() *VariableHeader {
	return nil
}
func (p *PingreqPacket) Payload() *Payload {
	return nil
}
func (p *PingreqPacket) WriteTo(w io.Writer) (n int64, err error) {
	m, err := w.Write(p.FixedHeader().Bytes())
	n = int64(m)
	return
}

func ParsePingreqPacketFromReader(r io.Reader, first byte) (p *PingreqPacket, err error) {
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
	p = &PingreqPacket{}
	return

}
func ParsePingreqPacket(b []byte) (p *PingreqPacket, err error) {
	if len(b) < 2 {
		err = ErrLessData
		return
	}
	if byte(TypePINGREQ) != b[0]>>4 {
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

	p = &PingreqPacket{}
	return
}
