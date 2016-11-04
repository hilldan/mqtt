package packet

import "errors"

// Integer data values
// Integer data values are 16 bits in big-endian order: the high order byte precedes the lower order byte.
// This means that a 16-bit word is presented on the network as Most Significant Byte (MSB), followed by
// Least Significant Byte (LSB).
type Integer uint16

func (i Integer) Bytes() []byte {
	return []byte{byte(i >> 8), byte(i)}
}
func IntegerFrom(high, low byte) Integer {
	return Integer(high)<<8 | Integer(low)
}

// UTF-8 encoded strings
// Text fields in the Control Packets described later are encoded as UTF-8 strings. UTF-8 [RFC3629] is an
// efficient encoding of Unicode [Unicode] characters that optimizes the encoding of ASCII characters in
// support of text-based communications.
// Each of these strings is prefixed with a two byte length field that gives the number of bytes in a UTF-8
// encoded string itself, as illustrated in Figure 1.1 Structure of UTF-8 encoded strings below. Consequently
// there is a limit on the size of a string that can be passed in one of these UTF-8 encoded string
// components; you cannot use a string that would encode to more than 65535 bytes.
type String string

func (s String) Bytes() []byte {
	if s == "" {
		return nil
	}
	b := []byte(s)
	if len(b) > 0xffff {
		b = b[:0xffff-1]
	}
	r := make([]byte, len(b)+2)
	copy(r, Integer(len(b)).Bytes())
	copy(r[2:], b)
	return r
}
func StringFrom(b []byte) String {
	if len(b) < 2 {
		return ""
	}
	l := int(IntegerFrom(b[0], b[1]))
	if len(b) < l+2 {
		return ""
	}
	return String(b[2 : 2+l])
}

type Bool bool

func (b Bool) Byte() byte {
	if b {
		return 1
	}
	return 0
}

type Bit2 byte

func (bit Bit2) Bits() (high, low byte) {
	high = byte(bit >> 1)
	low = byte(bit & 1)
	return
}
func Bit2From(high, low byte) Bit2 {
	return Bit2(high<<1 | low)
}

type Bit4 byte

func (bit4 Bit4) String() string {
	switch bit4 {
	case TypeCONNECT:
		return "TypeCONNECT"
	case TypeCONNACK:
		return "TypeCONNACK"
	case TypePUBLISH:
		return "TypePUBLISH"
	case TypePUBACK:
		return "TypePUBACK"
	case TypePUBREC:
		return "TypePUBREC"
	case TypePUBREL:
		return "TypePUBREL"
	case TypePUBCOMP:
		return "TypePUBCOMP"
	case TypeSUBSCRIBE:
		return "TypeSUBSCRIBE"
	case TypeSUBACK:
		return "TypeSUBACK"
	case TypeUNSUBSCRIBE:
		return "TypeUNSUBSCRIBE"
	case TypeUNSUBACK:
		return "TypeUNSUBACK"
	case TypePINGREQ:
		return "---------------------------------TypePINGREQ"
	case TypePINGRESP:
		return "---------------------------------TypePINGRESP"
	case TypeDISCONNECT:
		return "TypeDISCONNECT"
	}
	return "Invalid type"
}

var (
	ErrLessData         = errors.New("Data is not enough")
	ErrControlType      = errors.New("Control type unmatched")
	ErrFixedHeaderFlags = errors.New("FixedHeader's flags unmatched")
	ErrRemainLength     = errors.New("Remaining length unmatched")
	ErrProtocol         = errors.New("Protolcol is not MQTT 3.1.1")
	ErrClientId         = errors.New("client id invalid")
)
