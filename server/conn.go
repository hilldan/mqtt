package server

import (
	"encoding/json"
	"hilldan/mqtt"
	"hilldan/mqtt/packet"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Network Connection:
// A construct provided by the underlying transport protocol that is being used by MQTT.
// It connects the Client to the Server.
// It provides the means to send an ordered, lossless, stream of bytes in both directions.
type mqttConn struct {
	clientId string
	packetId uint32 //unique, convert into uint16

	//comunication between server and client
	cnn     net.Conn
	readch  chan mqtt.PacketReaded
	writech chan packet.ControlPacketer
	exitch  chan struct{}

	//session management
	session *mqtt.Session

	//keepalive
	deadline time.Duration
	pingch   chan struct{} //chan to indicate something come from client

	//close status
	dead  bool
	deadl sync.Mutex
}

func (c *mqttConn) read() {
	for {
		select {
		case <-c.exitch:
			goto exit
		default:
			p, err := packet.ParsePacket(c.cnn)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				goto exit
			}
			c.readch <- mqtt.PacketReaded{
				P:   p,
				Err: err,
			}
		}
	}
exit:
	close(c.readch)
	log.Printf("read no leak")
}

func (c *mqttConn) write() {
	for {
		select {
		case <-c.exitch:
			goto exit
		case p := <-c.writech:
			_, err := p.WriteTo(c.cnn)
			if err != nil {
				c.cnn.Close()
				goto exit
			}
		}
	}
exit:
	log.Printf("write no leak")
}

func (c *mqttConn) Close(cause string) {
	c.closeConn(cause, true)
}

func (c *mqttConn) closeConn(cause string, session bool) {
	log.Println("mqtt conn closed:", cause)
	c.deadl.Lock()
	defer c.deadl.Unlock()
	if c.dead {
		return
	}
	go listener.OnDisconnected()
	c.dead = true
	c.cnn.Close()
	close(c.exitch)
	close(c.pingch)
	if session {
		c.session.Save(KeySession, c.clientId, persister)
	}
	ConnRegistry.Remove(c.clientId)
}

//initConn wait for the first connect packet coming, handle it.
func (c *mqttConn) initConn() (p *packet.ConnectPacket) {
	select {
	case <-time.After(10e9):
		c.closeConn("waitting for connect packet timeout", false)
	case pr := <-c.readch:
		ack := new(packet.ConnackPacket)
		if pr.Err == packet.ErrProtocol {
			ack.Code = packet.CodeConnackRefusedProtocol
			ack.WriteTo(c.cnn) //sync write
			c.closeConn("protocol err", false)
			return
		}
		if pr.Err != nil {
			c.closeConn("parse the first connect packet err:"+pr.Err.Error(), false)
			return
		}
		if pr.P == nil {
			c.closeConn("connect fail", false)
			return
		}
		if pr.P.ControlType() != packet.TypeCONNECT {
			c.closeConn("the first packet is not a connect packet", false)
			return
		}

		p = pr.P.(*packet.ConnectPacket)
		if bool(p.UserNameFlag) && authCheck != nil && !authCheck(string(p.UserName), string(p.Password)) {
			ack.Code = packet.CodeConnackRefusedUnauthorized
			ack.WriteTo(c.cnn)
			c.closeConn("auth fail", false)
			p = nil
			return
		}

		c.clientId = string(p.ClientId)
		c.deadline = time.Second * time.Duration(p.KeepAlive)

		go c.write()
		go c.keepalive()

		if c.initSession() {
			c.publishOld(bool(p.CleanSession))
		}

		if p.CleanSession {
			ack.AckFlags = 1
		}
		ack.Code = packet.CodeConnackAccepted
		c.writech <- ack

		ConnRegistry.Add(c.clientId, c)

	}
	return
}
func (c *mqttConn) initSession() bool {
	data, err := persister.Read(KeySession, c.clientId)
	if err != nil {
		log.Printf("get session by '%s' fail: %v", c.clientId, err)
	}

	if len(data) == 0 {
		c.session = mqtt.NewSession()
		return false
	}

	s := new(mqtt.Session)
	err = json.Unmarshal(data, s)
	if err != nil {
		log.Printf("'%s' session data invalid", string(data))
		c.session = mqtt.NewSession()
		return false
	}
	s.MustInit()
	c.session = s
	return true
}

// publish send packet from server to client
func (c *mqttConn) publish(p packet.PublishPacket) {
	if p.Qos != packet.QoS0 {
		p.PacketId = packet.Integer(atomic.AddUint32(&c.packetId, 1))
	}
	p.Dup = false
	c.writech <- &p
	if p.Qos == packet.QoS0 {
		return
	}

	p.Dup = true
	c.session.AddPubOut(p.PacketId, p)
}

// publishOld extract unacknowledged packets from session and resend them to the peer.
func (c *mqttConn) publishOld(clearSession bool) {
	old := c.session.ResetPubOut()
	if clearSession {
		persister.Delete(KeySession, c.clientId)
		c.session = mqtt.NewSession()
	}
	max := packet.Integer(0)
	for _, v := range old {
		if v.PacketId > max {
			max = v.PacketId
		}
		c.writech <- &v
		c.session.AddPubOut(v.PacketId, v)
	}
	atomic.AddUint32(&c.packetId, uint32(max)+1) //keep unique
}

// func (c *mqttConn) republish() {
// 	tk := time.NewTicker(10e9)
// 	for {
// 		select {
// 		case <-tk.C:
// 			for _, v := range c.session.PubOut {
// 				c.writech <- &v
// 			}
// 		case <-c.exitch:
// 			tk.Stop()
// 			goto exit
// 		}
// 	}
// exit:
// 	log.Printf("republish no leak")
// }

func (c *mqttConn) subscribe(p packet.SubscribePacket) {
	l := len(p.TopicFilters)
	ack := &packet.SubackPacket{
		PacketId: p.PacketId,
		Code:     make([]byte, l),
	}
	if l == 0 {
		c.writech <- ack
		return
	}

	old := c.session.GetSubscription()
	ll := len(old)

	b := make([]packet.TopicFilter, l+ll)
	copy(b, old)
	for i, v := range p.TopicFilters {
		path, err := WildcardRegistry.Get(string(v.Topic))
		if err != nil {
			ack.Code[i] = packet.CodeSubackFailure
			continue
		}
		var add bool
		for k, vv := range old {
			path2, _ := WildcardRegistry.Get(string(vv.Topic))
			if _, relate := compare(path, path2); relate {
				b[k] = v
				add = true
				break
			}
		}
		if !add {
			b[ll] = v
			ll++
		}
		RetainRegistry.Publish(v, c)

		ack.Code[i] = byte(v.Qos)
	}
	c.session.SetSubscription(b[:ll])
	c.writech <- ack
}

func (c *mqttConn) keepalive() {
	//A Keep Alive value of zero (0) has the effect of turning off the keep alive mechanism
	if c.deadline == 0 {
		return
	}
	f := func() {
		c.closeConn("keepalive timeout", true)
	}
	time15 := c.deadline * 15 / 10
	tm := time.AfterFunc(time15, f)
	for range c.pingch {
		tm.Stop()
		tm = time.AfterFunc(time15, f)
	}
	tm.Stop()
	log.Printf("keepalive no leak")
}

/*
Todo:
1.
*/
