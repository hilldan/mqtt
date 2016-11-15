package server

import (
	"hilldan/mqtt"
	"hilldan/mqtt/connection"
	"hilldan/mqtt/packet"
	"log"
	"net"
	"time"
)

var (
	persister mqtt.Persister
	authCheck func(user, passwd string) bool
	listener  mqtt.EventListener
)

// TCP ports 8883 and 1883 are registered with IANA for MQTT TLS and non TLS communication
// respectively.

// A program or device that acts as an intermediary between Clients
// which publish Application Messages
// and Clients which have made Subscriptions. A Server
// Accepts Network Connections from Clients.
// Accepts Application Messages published by Clients.
// Processes Subscribe and Unsubscribe requests from Clients.
// Forwards Application Messages that match Client Subscriptions.
func RunMQTT(server connection.Serverer, persist mqtt.Persister) {
	persister = persist
	if persister == nil {
		panic("persisiter is nil")
	}
	if listener == nil {
		listener = mqtt.DefaultListener{}
	}
	RetainRegistry = NewRetainRegistry()
	server.Run(handler)
}

// SetPersister assign a persister to server.
func SetPersister(p mqtt.Persister) {
	persister = p
}

// SetAuthFunc assign a user authentication method to server that called
// when the connection has been established
func SetAuthFunc(f func(user, passwd string) bool) {
	authCheck = f
}

func SetEventListener(l mqtt.EventListener) {
	listener = l
}

// Publish send pub to the client specified by clientId.
func Publish(pub packet.PublishPacket, clientId string) {
	c, ok := ConnRegistry.Get(clientId)
	if !ok {
		return
	}
	c.publish(pub)
}

func handler(cnn net.Conn) {
	//init
	const N = 10
	c := &mqttConn{
		cnn:     cnn,
		readch:  make(chan mqtt.PacketReaded, N),
		writech: make(chan packet.ControlPacketer, N),
		exitch:  make(chan struct{}),
		pingch:  make(chan struct{}, N),
	}
	go c.read()

	pc := c.initConn()
	if pc == nil {
		return
	}

	go func() {
		if err := listener.OnConnected(*pc); err != nil {
			time.Sleep(1e6)
			c.closeConn(err.Error(), false)
		}
	}()
	//handle all the packet
	for pr := range c.readch {
		if pr.Err != nil {
			lastwill(pc, string(c.clientId))
			c.closeConn(pr.Err.Error(), true)
			goto exit
		}
		if pr.P == nil {
			c.closeConn("connect fail", true)
			goto exit
		}
		go handlePacket(pr.P, c, pc)
	}
exit:
	log.Printf("handler no leak")
}

func lastwill(pc *packet.ConnectPacket, excludeId string) {
	if !pc.WillFlag {
		return
	}
	pub := packet.PublishPacket{
		Qos:                pc.WillQoS,
		Retain:             pc.WillRetain,
		TopicName:          pc.WillTopic,
		ApplicationMessage: pc.WillMessage,
	}
	if pub.Retain {
		RetainRegistry.Add(string(pub.TopicName), pub)
	}
	ConnRegistry.Publish(pub, excludeId)
}

func handlePacket(p packet.ControlPacketer, c *mqttConn, pc *packet.ConnectPacket) {
	if c.deadline > 0 {
		c.pingch <- struct{}{}
	}
	switch p.ControlType() {
	case packet.TypeCONNECT: //a second CONNECT Packet sent from a Client as a protocol violation
		c.closeConn("second connect packet", true)

	// case packet.TypeCONNACK:
	case packet.TypePUBLISH:
		pk := p.(*packet.PublishPacket)
		//response
		switch pk.Qos {
		case packet.QoS0:
		case packet.QoS1:
			c.writech <- &packet.PubackPacket{PacketId: pk.PacketId}
		case packet.QoS2:
			c.writech <- &packet.PubrecPacket{PacketId: pk.PacketId}
			if bool(pk.Dup) && c.session.GetPubIn(pk.PacketId) {
				return
			}
			c.session.AddPubIn(pk.PacketId)
		}

		// save and distribute
		if pk.Retain {
			RetainRegistry.Add(string(pk.TopicName), *pk)
		}
		ConnRegistry.Publish(*pk, c.clientId)
		go listener.OnPublishReceived(*pk)

	case packet.TypePUBACK:
		pk := p.(*packet.PubackPacket)
		c.session.RemovePubOut(pk.PacketId)

	case packet.TypePUBREC:
		pk := p.(*packet.PubrecPacket)
		c.writech <- &packet.PubrelPacket{PacketId: pk.PacketId}
		c.session.RemovePubOut(pk.PacketId)

	case packet.TypePUBREL:
		pk := p.(*packet.PubrelPacket)
		c.session.RemovePubIn(pk.PacketId)
		c.writech <- &packet.PubcompPacket{PacketId: pk.PacketId}

	case packet.TypePUBCOMP:
		// do nothing
		// pk := p.(*packet.PubcompPacket)

	case packet.TypeSUBSCRIBE:
		pk := p.(*packet.SubscribePacket)
		c.subscribe(*pk)
		go listener.OnSubscribeSuccess(pk.TopicFilters)

	// case packet.TypeSUBACK:
	case packet.TypeUNSUBSCRIBE:
		pk := p.(*packet.UnsubscribePacket)
		c.session.Unsubscription(pk.TopicFilter)
		c.writech <- &packet.UnsubackPacket{PacketId: pk.PacketId}
		go listener.OnUnsubscribeSuccess(pk.TopicFilter)

	// case packet.TypeUNSUBACK:
	case packet.TypePINGREQ:
		c.writech <- &packet.PingrespPacket{}

	// case packet.TypePINGRESP:
	case packet.TypeDISCONNECT:
		pc.WillFlag = false
		c.closeConn("disconnect", true)

	default:
		c.closeConn("invalid packet", true)
	}
}
