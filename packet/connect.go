package packet

import (
	"bytes"
	"io"
)

const (
	ProtocolName   = String("MQTT")
	ProtocolLevel  = byte(4)
	VaryHeadLength = 10
)

// CONNECT – Client requests a connection to a Server
// After a Network Connection is established by a Client to a Server, the first Packet sent from the Client to
// the Server MUST be a CONNECT Packet
type ConnectPacket struct {
	remainLength int
	// ConnectFlags
	UserNameFlag Bool
	PasswdFlag   Bool
	WillRetain   Bool
	WillQoS      Bit2
	WillFlag     Bool
	CleanSession Bool
	KeepAlive    Integer

	//payload
	ClientId    String
	WillTopic   String
	WillMessage String
	UserName    String
	Password    String
}

func (p *ConnectPacket) ControlType() Bit4 { return TypeCONNECT }
func (p *ConnectPacket) FixedHeader() *FixedHeader {
	if p.remainLength == 0 {
		p.Payload()
	}
	return &FixedHeader{
		ControlType: TypeCONNECT,
		Flags:       0,
		Remaining:   p.remainLength,
	}
}
func (p *ConnectPacket) VariableHeader() *VariableHeader {
	vary := make([]byte, VaryHeadLength)
	copy(vary, ProtocolName.Bytes())
	vary[6] = ProtocolLevel
	vary[7] = p.connectFlags()
	copy(vary[8:], p.KeepAlive.Bytes())
	return &VariableHeader{
		Vary: vary,
	}
}

// The payload of the CONNECT Packet contains one or more length-prefixed fields, whose presence is
// determined by the flags in the variable header. These fields, if present, MUST appear in the order Client
// Identifier, Will Topic, Will Message, User Name, Password
func (p *ConnectPacket) Payload() *Payload {
	b := p.ClientId.Bytes()
	if p.WillFlag {
		b = append(b, p.WillTopic.Bytes()...)
		b = append(b, p.WillMessage.Bytes()...)
	}
	if p.UserNameFlag {
		b = append(b, p.UserName.Bytes()...)
	}
	if p.PasswdFlag {
		b = append(b, p.Password.Bytes()...)
	}
	p.remainLength = VaryHeadLength + len(b)
	return &Payload{
		Content: b,
	}
}
func (p *ConnectPacket) WriteTo(w io.Writer) (n int64, err error) {
	payload := p.Payload().Content
	varyHead := p.VariableHeader().Vary
	fixHead := p.FixedHeader().Bytes()
	var buf bytes.Buffer
	buf.Write(fixHead)
	buf.Write(varyHead)
	buf.Write(payload)
	return buf.WriteTo(w)
}

func (p *ConnectPacket) connectFlags() byte {
	qosh, qosl := p.WillQoS.Bits()
	return p.UserNameFlag.Byte()<<7 | p.PasswdFlag.Byte()<<6 | p.WillRetain.Byte()<<5 | qosh<<4 | qosl<<3 | p.WillFlag.Byte()<<2 | p.CleanSession.Byte()<<1
}

// ParseConnectPacket parse the data in bytes into a ConnectPacket.
func ParseConnectPacket(b []byte) (p *ConnectPacket, err error) {
	if len(b) < 14 {
		err = ErrLessData
		return
	}

	//parse FixedHeader
	if 0 != byte(b[0]&0x0f) {
		err = ErrFixedHeaderFlags
		return
	}
	remainl, remainConent := decodeRemainLength(b[1:])
	if remainl < VaryHeadLength || len(remainConent) < remainl {
		err = ErrRemainLength
		return
	}

	//parse VariableHeader
	if !bytes.Equal(ProtocolName.Bytes(), remainConent[:6]) || ProtocolLevel != remainConent[6] {
		err = ErrProtocol
		return
	}
	userNameFlag := remainConent[7]&0x80 != 0
	passwdFlag := remainConent[7]&0x40 != 0
	willRetain := remainConent[7]&0x20 != 0
	willQoS := Bit2(remainConent[7] & 0x18 >> 3)
	willFlag := remainConent[7]&0x04 != 0
	cleanSession := remainConent[7]&0x02 != 0
	keepAlive := IntegerFrom(remainConent[8], remainConent[9])

	if !userNameFlag {
		passwdFlag = false
	}
	if !willFlag {
		willRetain = false
	}

	//parse payload
	start := 10
	clientId := StringFrom(remainConent[start:])
	start += len(clientId) + 2
	var willTopic, willMessage String
	if willFlag {
		willTopic = StringFrom(remainConent[start:])
		start += len(willTopic) + 2
		willMessage = StringFrom(remainConent[start:])
		start += len(willMessage) + 2
	}
	var userName, passwd String
	if userNameFlag {
		userName = StringFrom(remainConent[start:])
		start += len(userName) + 2
	}
	if passwdFlag {
		passwd = StringFrom(remainConent[start:])
		start += len(passwd) + 2
	}

	p = &ConnectPacket{
		remainLength: remainl,
		UserNameFlag: Bool(userNameFlag),
		PasswdFlag:   Bool(passwdFlag),
		WillRetain:   Bool(willRetain),
		WillQoS:      willQoS,
		WillFlag:     Bool(willFlag),
		CleanSession: Bool(cleanSession),
		KeepAlive:    keepAlive,
		ClientId:     clientId,
		WillTopic:    willTopic,
		WillMessage:  willMessage,
		UserName:     userName,
		Password:     passwd,
	}
	return
}

// ParseConnectPacketFromReader read from a reader and parse its data into a ConnectPacket.
func ParseConnectPacketFromReader(r io.Reader, first byte) (p *ConnectPacket, err error) {
	//fixed header
	if 0 != byte(first&0x0f) {
		err = ErrFixedHeaderFlags
		return
	}
	buf := make([]byte, 4)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return
	}
	rl, remainbuf := decodeRemainLength(buf)
	if rl < VaryHeadLength {
		err = ErrRemainLength
		return
	}

	//variable header
	buf = make([]byte, VaryHeadLength)
	n := copy(buf, remainbuf)
	_, err = io.ReadFull(r, buf[n:])
	if err != nil {
		return
	}
	if !bytes.Equal(ProtocolName.Bytes(), buf[:6]) || ProtocolLevel != buf[6] {
		err = ErrProtocol
		return
	}
	userNameFlag := buf[7]&0x80 != 0
	passwdFlag := buf[7]&0x40 != 0
	willRetain := buf[7]&0x20 != 0
	willQoS := Bit2(buf[7] & 0x18 >> 3)
	willFlag := buf[7]&0x04 != 0
	cleanSession := buf[7]&0x02 != 0
	keepAlive := IntegerFrom(buf[8], buf[9])

	if !userNameFlag {
		passwdFlag = false
	}
	if !willFlag {
		willRetain = false
	}

	//payload
	clientId, err := readStringFrom(r)
	if err != nil {
		return
	}
	if clientId == "" {
		err = ErrClientId
		return
	}
	var willTopic, willMessage String
	if willFlag {
		willTopic, err = readStringFrom(r)
		if err != nil {
			return
		}
		willMessage, err = readStringFrom(r)
		if err != nil {
			return
		}
	}
	var userName, passwd String
	if userNameFlag {
		userName, err = readStringFrom(r)
		if err != nil {
			return
		}
	}
	if passwdFlag {
		passwd, err = readStringFrom(r)
		if err != nil {
			return
		}
	}
	p = &ConnectPacket{
		remainLength: rl,
		UserNameFlag: Bool(userNameFlag),
		PasswdFlag:   Bool(passwdFlag),
		WillRetain:   Bool(willRetain),
		WillQoS:      willQoS,
		WillFlag:     Bool(willFlag),
		CleanSession: Bool(cleanSession),
		KeepAlive:    keepAlive,
		ClientId:     clientId,
		WillTopic:    willTopic,
		WillMessage:  willMessage,
		UserName:     userName,
		Password:     passwd,
	}
	return

}

func readStringFrom(r io.Reader) (s String, err error) {
	var b [2]byte
	_, err = io.ReadFull(r, b[:])
	if err != nil {
		return
	}
	sl := IntegerFrom(b[0], b[1])
	bb := make([]byte, int(sl))
	_, err = io.ReadFull(r, bb)
	if err != nil {
		return
	}
	s = String(bb)
	return
}
