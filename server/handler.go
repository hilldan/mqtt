package server

import (
	"hilldan/mqtt"
	"hilldan/mqtt/connection"
	"hilldan/mqtt/packet"
	"log"
	"net"
	"sync/atomic"
)

var (
	persister mqtt.Persister
	packetId  uint32 //unique, convert into uint16
	authCheck func(user, passwd string) bool
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
func RunMQTT(server connection.Serverer, persist mqtt.Persister, auth func(user, passwd string) bool) {
	persister = persist
	RetainRegistry = NewRetainRegistry()
	authCheck = auth
	server.Run(handler)
}

func SetPersister(p mqtt.Persister) {
	persister = p
}

func handler(cnn net.Conn) {
	//init
	const N = 10
	c := &mqttConn{
		cnn:     cnn,
		readch:  make(chan mqtt.PacketReaded, N),
		writech: make(chan packet.ControlPacketer, N),
		exitch:  make(chan struct{}),
		puback:  make(map[uint16]chan struct{}),
		pingch:  make(chan struct{}, N),
	}
	go c.read()

	pc := c.initConn()
	if pc == nil {
		return
	}
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
		}

		// don't ignore any packet from client
		// if PacketIdRegistry.Ignore(c.clientId, pk.PacketId) {
		// 	return
		// }

		//save and distribute
		pk.PacketId = packet.Integer(atomic.AddUint32(&packetId, 1))
		if pk.Retain {
			RetainRegistry.Add(string(pk.TopicName), *pk)
		}
		ConnRegistry.Publish(*pk, c.clientId)

	case packet.TypePUBACK:
		pk := p.(*packet.PubackPacket)
		c.closeTikerch(pk.PacketId)

	case packet.TypePUBREC:
		pk := p.(*packet.PubrecPacket)
		c.writech <- &packet.PubrelPacket{PacketId: pk.PacketId}

	case packet.TypePUBREL:
		pk := p.(*packet.PubrelPacket)
		c.writech <- &packet.PubcompPacket{PacketId: pk.PacketId}

	case packet.TypePUBCOMP:
		pk := p.(*packet.PubcompPacket)
		c.closeTikerch(pk.PacketId)

	case packet.TypeSUBSCRIBE:
		pk := p.(*packet.SubscribePacket)
		go c.subscribe(*pk)

	// case packet.TypeSUBACK:
	case packet.TypeUNSUBSCRIBE:
		pk := p.(*packet.UnsubscribePacket)
		c.session.Unsubscription(pk.TopicFilter)
		c.writech <- &packet.UnsubackPacket{PacketId: pk.PacketId}

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
