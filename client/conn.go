package client

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
	//comunication between server and client
	cnn     net.Conn
	readch  chan mqtt.PacketReaded
	writech chan packet.ControlPacketer
	exitch  chan struct{}

	//session management
	session *mqtt.Session

	//keepalive
	deadline time.Duration
	pingch   chan struct{}

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
		c.session.Save(KeySession, ClientId, persister)
	}
}

//initConn wait for the first connect packet coming, handle it.
func (c *mqttConn) initConn(clearSession bool) error {
	select {
	case <-time.After(10e9):
		c.closeConn("waiting for connack packet timeout", false)
		return mqtt.ErrTimeout
	case pr := <-c.readch:
		if pr.Err != nil {
			c.closeConn(pr.Err.Error(), false)
			return pr.Err
		}
		if pr.P == nil {
			c.closeConn("connect fail", false)
			return mqtt.ErrConnect
		}
		if pr.P.ControlType() != packet.TypeCONNACK {
			c.closeConn("the first packet is not a connack packet", false)
			return packet.ErrControlType
		}

		p := pr.P.(*packet.ConnackPacket)
		if p.Code == packet.CodeConnackRefusedUnauthorized || p.Code == packet.CodeConnackRefusedUserPasswd {
			c.closeConn("auth fail", false)
			return mqtt.ErrAuth
		}
		if p.Code != packet.CodeConnackAccepted {
			c.closeConn("connect refused", false)
			return mqtt.ErrConnect
		}
		// log.Printf("server sp value %d", p.AckFlags&0x01)
		if c.initSession() {
			c.publishOld(clearSession)
		}
		go c.keepalive()
	}
	return nil
}
func (c *mqttConn) initSession() bool {
	data, err := persister.Read(KeySession, ClientId)
	if err != nil {
		log.Printf("get session by '%s' fail: %v", ClientId, err)
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

func (c *mqttConn) keepalive() {
	//A Keep Alive value of zero (0) has the effect of turning off the keep alive mechanism
	if c.deadline == 0 {
		return
	}
	f1 := func() {
		c.writech <- &packet.PingreqPacket{}
	}
	f2 := func() {
		c.closeConn("keepalive timeout", true)
	}
	timeResp := c.deadline + 10*time.Second
	tm1 := time.AfterFunc(c.deadline, f1)
	tm2 := time.AfterFunc(timeResp, f2)

	for range c.pingch {
		tm1.Stop()
		tm2.Stop()
		tm1 = time.AfterFunc(c.deadline, f1)
		tm2 = time.AfterFunc(timeResp, f2)
	}
	tm1.Stop()
	tm2.Stop()
	log.Printf("keepalive no leak")
}

// Publish send packet from client to server.
func (c *mqttConn) Publish(p packet.PublishPacket) {
	if c.IsDead() {
		return
	}
	p.PacketId = packet.Integer(atomic.AddUint32(&PacketId, 1))
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
		persister.Delete(KeySession, ClientId)
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
	atomic.AddUint32(&PacketId, uint32(max)+1) //keep unique
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

// Subscribe send topic filters to server. When a suback received from server,
// the subject subscribed successfully will be saved at session.
func (c *mqttConn) Subscribe(filters []packet.TopicFilter) {
	if c.IsDead() {
		return
	}
	p := &packet.SubscribePacket{
		PacketId:     packet.Integer(atomic.AddUint32(&PacketId, 1)),
		TopicFilters: filters,
	}
	TopicFilterRegistry.AddSubs(uint16(p.PacketId), filters)
	c.writech <- p
}

func (c *mqttConn) handleSuback(p *packet.SubackPacket) {
	subs, ok := TopicFilterRegistry.GetRemoveSubs(uint16(p.PacketId))
	if !ok {
		return
	}
	if len(subs) != len(p.Code) {
		return
	}
	go listener.OnSubscribeSuccess(subs)
	tem := make([]packet.TopicFilter, len(subs))
	n := 0
	for i := 0; i < len(p.Code); i++ {
		if p.Code[i] == packet.CodeSubackFailure {
			continue
		}
		tem[n] = subs[i]
		n++
	}
	c.session.AddSubscription(tem[:n])
}

// Unsubscribe send command unsubscribe to server. When a unsuback received from server,
// the subject unsubscribed successfully will be modified at session.
func (c *mqttConn) Unsubscribe(ts []packet.String) {
	if c.IsDead() {
		return
	}
	p := &packet.UnsubscribePacket{
		PacketId:    packet.Integer(atomic.AddUint32(&PacketId, 1)),
		TopicFilter: ts,
	}
	TopicFilterRegistry.AddUnsubs(uint16(p.PacketId), ts)
	c.writech <- p
}
func (c *mqttConn) handleUnsuback(pid uint16) {
	unsbs, ok := TopicFilterRegistry.GetRemoveUnsubs(pid)
	if !ok {
		return
	}
	go listener.OnUnsubscribeSuccess(unsbs)
	c.session.Unsubscription(unsbs)
}

// Disconnect tell server to close connection.
func (c *mqttConn) Disconnect() {
	if c.IsDead() {
		return
	}
	c.writech <- &packet.DisconnectPacket{}
	time.Sleep(1e6)
	c.closeConn("disconnect", true)
}

// IsDead reports the connection is closed or not
func (c *mqttConn) IsDead() bool {
	c.deadl.Lock()
	b := c.dead
	c.deadl.Unlock()
	return b
}
