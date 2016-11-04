package server

import (
	"encoding/json"
	"hilldan/mqtt"
	"hilldan/mqtt/packet"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// Network Connection:
// A construct provided by the underlying transport protocol that is being used by MQTT.
// It connects the Client to the Server.
// It provides the means to send an ordered, lossless, stream of bytes in both directions.
type mqttConn struct {
	clientId string

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
	c.dead = true
	c.cnn.Close()
	close(c.exitch)
	close(c.pingch)
	c.closeAllTikerch()
	if session {
		c.session.Save(KeySession, c.clientId, persister)
	}
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
		if bool(p.UserNameFlag) && !authCheck(string(p.UserName), string(p.Password)) {
			ack.Code = packet.CodeConnackRefusedUnauthorized
			ack.WriteTo(c.cnn)
			c.closeConn("auth fail", false)
			return
		}
		c.clientId = string(p.ClientId)
		c.deadline = time.Second * time.Duration(p.KeepAlive)

		go c.write()
		go c.keepalive()

		isOld := c.initSession(p)
		if isOld {
			ack.AckFlags = 1
		}
		ack.Code = packet.CodeConnackAccepted
		c.writech <- ack

		ConnRegistry.Add(c.clientId, c)

	}
	return
}
func (c *mqttConn) initSession(p *packet.ConnectPacket) bool {
	data, err := persister.Read(KeySession, c.clientId)
	if err != nil {
		log.Printf("get session by '%s' fail: %v", c.clientId, err)
	}

	if len(data) == 0 {
		c.session = mqtt.NewSession()
		return false
	}
	if p.CleanSession {
		persister.Delete(KeySession, c.clientId)
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

// publish send packet from server to client
// be careful that leakage accur after connection closed
func (c *mqttConn) publish(p packet.PublishPacket) {
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
	log.Printf("publish %d no leak", p.PacketId)
}

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
	log.Printf("keepalive no leak")
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
