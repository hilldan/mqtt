package packet

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestRemainLength(t *testing.T) {
	var tcs = []struct {
		in  int
		out []byte
	}{
		{in: 0, out: []byte{0}},
		{in: 127, out: []byte{0x7f}},
		{in: 128, out: []byte{0x80, 0x01}},
		{in: 16383, out: []byte{0xff, 0x7f}},
		{in: 16384, out: []byte{0x80, 0x80, 0x01}},
		{in: 2097151, out: []byte{0xff, 0xff, 0x7f}},
		{in: 2097152, out: []byte{0x80, 0x80, 0x80, 0x01}},
		{in: 268435455, out: []byte{0xff, 0xff, 0xff, 0x7f}},

		{in: 321, out: []byte{0xc1, 0x02}},
	}
	for _, v := range tcs {
		b := encodeRemainLength(v.in)
		if !bytes.Equal(b, v.out) {
			t.Errorf("encodeRemainLength err, want %v, actual %v", v.out, b)
		}

		i, bb := decodeRemainLength(v.out)
		if i != v.in {
			t.Errorf("decodeRemainLength err, want %d, actual %d", v.in, i)
		}
		if len(bb) != 0 {
			t.Errorf("decodeRemainLength err, remainning want size 0,actual %d", len(bb))
		}

		//
		r := bytes.NewReader(v.out)
		l, err := decodeRemainLengthFromReader(r)
		if err != nil {
			t.Errorf("decodeRemainLengthFromReader err:", err)
		}
		if l != v.in {
			t.Errorf("decodeRemainLengthFromReader err: want %d actual %d", v.in, l)
		}
	}
}

func testPacket(t *testing.T, o ControlPacketer, f func(r io.Reader, first byte)) {
	done := make(chan struct{})
	r, w := io.Pipe()
	go func() {
		var first [1]byte
		_, err := r.Read(first[:])
		if err != nil {
			t.Errorf("read first byte err: %v", err)
		}
		f(r, first[0])
		close(done)
	}()
	_, err := o.WriteTo(w)
	if err != nil {
		t.Error(err)
	}
	<-done
}

func TestConnectPacket(t *testing.T) {
	o := &ConnectPacket{
		WillFlag:     true,
		UserNameFlag: true,
		PasswdFlag:   true,
		ClientId:     "sdi34",
		UserName:     "hhyy",
		Password:     "uhiij",
		WillTopic:    "xx",
		WillMessage:  "xxxyyyy",
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParseConnectPacketFromReader(r, first)
		if err != nil {
			t.Errorf("ParseConnectPacketFromReader err: %v", err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if *o != *p {
			t.Errorf("Parse Packet err")
		}
	})
}

func TestConnackPacket(t *testing.T) {
	o := &ConnackPacket{
		AckFlags: 1,
		Code:     4,
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParseConnackPacketFromReader(r, first)
		if err != nil {
			t.Errorf("Parse Packet From Reader err: %v", err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if *o != *p {
			t.Errorf("Parse Packet err")
		}
	})
}
func TestPublishPacket(t *testing.T) {
	o := &PublishPacket{
		Dup:                true,
		Qos:                1,
		Retain:             false,
		TopicName:          "topicName",
		PacketId:           134,
		ApplicationMessage: "message",
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParsePublishPacketFromReader(r, first)
		if err != nil {
			t.Errorf("Parse Packet From Reader err: %v", err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if *o != *p {
			t.Errorf("Parse Packet err")
		}
	})
}

func TestPubackPacket(t *testing.T) {
	o := &PubackPacket{
		PacketId: 13477,
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParsePubackPacketFromReader(r, first)
		if err != nil {
			t.Errorf("Parse Packet From Reader err: %v", err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if *o != *p {
			t.Errorf("Parse Packet err")
		}
	})
}

func TestPubrecPacket(t *testing.T) {
	o := &PubrecPacket{
		PacketId: 23,
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParsePubrecPacketFromReader(r, first)
		if err != nil {
			t.Errorf("Parse Packet From Reader err: %v", err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if *o != *p {
			t.Errorf("Parse Packet err")
		}
	})
}

func TestPubrelPacket(t *testing.T) {
	o := &PubrelPacket{
		PacketId: 23,
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParsePubrelPacketFromReader(r, first)
		if err != nil {
			t.Errorf("Parse Packet From Reader err: %v", err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if *o != *p {
			t.Errorf("Parse Packet err")
		}
	})
}

func TestPubcompPacket(t *testing.T) {
	o := &PubcompPacket{
		PacketId: 23,
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParsePubcompPacketFromReader(r, first)
		if err != nil {
			t.Errorf("Parse Packet From Reader err: %v", err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if *o != *p {
			t.Errorf("Parse Packet err")
		}
	})
}

func TestSubscribePacket(t *testing.T) {
	o := &SubscribePacket{
		PacketId: 23,
		TopicFilters: []TopicFilter{
			{"xx/a", 1},
			{"yy/b", 2},
			{"zz/c", 0},
		},
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParseSubscribePacketFromReader(r, first)
		if err != nil {
			t.Errorf("Parse Packet From Reader err: %v", err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if o.PacketId != p.PacketId || o.remainl != p.remainl {
			t.Errorf("Parse Packet err")
		}
		for k, v := range o.TopicFilters {
			if v != p.TopicFilters[k] {
				t.Errorf("Parse Packet err: want %v actual %v", v, p.TopicFilters[k])
			}
		}
	})
}

func TestSubackPacket(t *testing.T) {
	o := &SubackPacket{
		PacketId: 23,
		Code:     []byte{1, 2, 0, 128},
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParseSubackPacketFromReader(r, first)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if o.PacketId != p.PacketId || o.remainl != p.remainl {
			t.Errorf("Parse Packet err")
		}
		if !bytes.Equal(o.Code, p.Code) {
			t.Errorf("parse err: want %v actual %v", o.Code, p.Code)
		}
	})
}

func TestUnsubscribePacket(t *testing.T) {
	o := &UnsubscribePacket{
		PacketId:    23,
		TopicFilter: []String{"xx/a", "yy/b", "zz/c/d"},
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParseUnsubscribePacketFromReader(r, first)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if o.PacketId != p.PacketId || o.remainl != p.remainl {
			t.Errorf("Parse Packet err")
		}
		if len(o.TopicFilter) != len(p.TopicFilter) {
			t.Errorf("parse err: topic filter size want %d actual %d", len(o.TopicFilter), len(p.TopicFilter))
		}
		for i := 0; i < len(o.TopicFilter); i++ {
			if o.TopicFilter[i] != p.TopicFilter[i] {
				t.Errorf("parse err: want '%s', actual '%s'", o.TopicFilter[i], p.TopicFilter[i])
			}
		}
	})
}

func TestUnsubackPacket(t *testing.T) {
	o := &UnsubackPacket{
		PacketId: 23,
	}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParseUnsubackPacketFromReader(r, first)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
		if o.PacketId != p.PacketId {
			t.Errorf("Parse Packet err")
		}
	})
}

func TestPingPacket(t *testing.T) {
	o := &PingreqPacket{}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParsePingreqPacketFromReader(r, first)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
	})
	oo := &PingrespPacket{}
	testPacket(t, oo, func(r io.Reader, first byte) {
		p, err := ParsePingrespPacketFromReader(r, first)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(*oo)
		fmt.Println(*p)
	})
}

func TestDisconnectPacket(t *testing.T) {
	o := &DisconnectPacket{}
	testPacket(t, o, func(r io.Reader, first byte) {
		p, err := ParseDisconnectPacketFromReader(r, first)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(*o)
		fmt.Println(*p)
	})
}
