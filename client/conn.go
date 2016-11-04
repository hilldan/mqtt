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
	rwl     sync.RWMutex //protect sesssion

	//publish status
	puback  map[uint16]chan struct{} //packetId->chan to stop tiker
	pubackl sync.RWMutex

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
	c.dead = true
	c.cnn.Close()
	close(c.exitch)
	close(c.pingch)
	c.closeAllTikerch()
	if session {
		c.session.Save(KeySession, ClientId, persister)
	}
}

//initConn wait for the first connect packet coming, handle it.
func (c *mqttConn) initConn() error {
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
			return mqtt.ErrAuth
		}
		if p.Code != packet.CodeConnackAccepted {
			c.closeConn("connect refused", false)
			return mqtt.ErrConnect
		}
		// log.Printf("server sp value %d", p.AckFlags&0x01)

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
	log.Printf("keepalive no leak")
}

// Publish send packet from client to server.
// It will block  for QOS>0 until acknowledged by server
func (c *mqttConn) Publish(p packet.PublishPacket) {
	p.PacketId = packet.Integer(atomic.AddUint32(&PacketId, 1))
	c.writech <- &p
	if p.Qos == packet.QoS0 {
		return
	}

	p.Dup = true
	c.session.AddPubOut(p.PacketId, p)

	tk := time.NewTicker(10e9)
	ch := c.getTikerch(p.PacketId)
	for {
		select {
		case <-tk.C:
			c.writech <- &p
		case <-ch:
			tk.Stop()
			c.session.RemovePubOut(p.PacketId)
			goto exit
		}
	}
exit:
	log.Printf("Publish %d no leak", p.PacketId)
}

// Subscribe send topic filters to server. When a suback received from server,
// the subject subscribed successfully will be saved at session.
func (c *mqttConn) Subscribe(filters []packet.TopicFilter) {
	p := &packet.SubscribePacket{
		PacketId:     packet.Integer(atomic.AddUint32(&PacketId, 1)),
		TopicFilters: filters,
	}
	TopicFilterRegistry.AddSubs(uint16(p.PacketId), filters)
	c.writech <- p
}

func (c *mqttConn) handleSuback(p *packet.SubackPacket) {
	subs, ok := TopicFilterRegistry.GetSubs(uint16(p.PacketId))
	if !ok {
		return
	}
	if len(subs) != len(p.Code) {
		return
	}
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
	TopicFilterRegistry.RemoveSubs(uint16(p.PacketId))
}

// Unsubscribe send command unsubscribe to server. When a unsuback received from server,
// the subject unsubscribed successfully will be modified at session.
func (c *mqttConn) Unsubscribe(ts []packet.String) {
	p := &packet.UnsubscribePacket{
		PacketId:    packet.Integer(atomic.AddUint32(&PacketId, 1)),
		TopicFilter: ts,
	}
	TopicFilterRegistry.AddUnsubs(uint16(p.PacketId), ts)
	c.writech <- p
}
func (c *mqttConn) handleUnsuback(pid uint16) {
	unsbs, ok := TopicFilterRegistry.GetUnsubs(pid)
	if !ok {
		return
	}
	c.session.Unsubscription(unsbs)
	TopicFilterRegistry.RemoveUnsubs(pid)
}

// Disconnect tell server to close connection.
func (c *mqttConn) Disconnect() {
	c.writech <- &packet.DisconnectPacket{}
	c.closeConn("disconnect", true)
}

func (c *mqttConn) getTikerch(pid packet.Integer) (ch chan struct{}) {
	c.pubackl.Lock()
	defer c.pubackl.Unlock()
	ch, ok := c.puback[uint16(pid)]
	if ok {
		return
	}
	ch = make(chan struct{})
	c.puback[uint16(pid)] = ch
	return
}
func (c *mqttConn) closeTikerch(pid packet.Integer) {
	c.pubackl.Lock()
	defer c.pubackl.Unlock()
	ch, ok := c.puback[uint16(pid)]
	if !ok {
		return
	}
	close(ch)
	delete(c.puback, uint16(pid))
}
func (c *mqttConn) closeAllTikerch() {
	c.pubackl.Lock()
	defer c.pubackl.Unlock()
	for _, v := range c.puback {
		close(v)
	}
	c.puback = make(map[uint16]chan struct{})
}
