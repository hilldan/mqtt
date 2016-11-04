package client

import (
	"hilldan/mqtt"
	"hilldan/mqtt/connection"
	"hilldan/mqtt/packet"
	"log"
	"time"
)

const (
	KeySession = "mq:cs"
)

var (
	persister     mqtt.Persister
	ClientId      string
	PacketId      uint32 //convert into packet.Integer
	handleMessage func(msg string)
)

func SetPersister(p mqtt.Persister) {
	persister = p
}

// TCP ports 8883 and 1883 are registered with IANA for MQTT TLS and non TLS communication
// respectively.

// A program or device that uses MQTT.
// A Client always establishes the Network Connection to the Server.
// It can
// Publish Application Messages that other Clients might be interested in.
// Subscribe to request Application Messages that it is interested in receiving.
// Unsubscribe to remove a request for Application Messages.
// Disconnect from the Server.
func RunMQTT(client connection.Clienter, persist mqtt.Persister, p *packet.ConnectPacket, handler func(msg string)) (cnn *mqttConn, err error) {
	conn, err := client.Dial()
	if err != nil {
		return
	}
	persister = persist
	ClientId = string(p.ClientId)
	handleMessage = handler

	const N = 10

	cnn = &mqttConn{
		cnn:      conn,
		readch:   make(chan mqtt.PacketReaded, N),
		writech:  make(chan packet.ControlPacketer, N),
		exitch:   make(chan struct{}),
		puback:   make(map[uint16]chan struct{}),
		pingch:   make(chan struct{}, N),
		deadline: time.Second * time.Duration(p.KeepAlive),
	}
	go cnn.write()
	go cnn.read()

	old := cnn.initSession()
	p.CleanSession = packet.Bool(!old)
	cnn.writech <- p
	err = cnn.initConn()
	if err != nil {
		return
	}
	go readPacket(cnn)

	return
}

//read and handle all the packet
func readPacket(c *mqttConn) {
	for pr := range c.readch {
		if pr.Err != nil {
			c.closeConn(pr.Err.Error(), true)
			break
		}
		if pr.P == nil {
			c.closeConn("connect fail", true)
			break
		}
		go handlePacket(pr.P, c)
	}
	log.Printf("handler no leak")
}
func handlePacket(p packet.ControlPacketer, c *mqttConn) {
	if c.deadline > 0 {
		c.pingch <- struct{}{}
	}
	switch p.ControlType() {
	// case packet.TypeCONNECT:
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
		// ignore any publish packet that has been handled from server
		if pk.Qos != packet.QoS0 && PacketIdRegistry.Ignore(pk.PacketId, packet.TypePUBLISH) {
			return
		}
		//do something
		handleMessage(string(pk.ApplicationMessage))

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

	// case packet.TypeSUBSCRIBE:
	case packet.TypeSUBACK:
		pk := p.(*packet.SubackPacket)
		c.handleSuback(pk)

	// case packet.TypeUNSUBSCRIBE:
	case packet.TypeUNSUBACK:
		pk := p.(*packet.UnsubackPacket)
		c.handleUnsuback(uint16(pk.PacketId))

	// case packet.TypePINGREQ:
	case packet.TypePINGRESP:
		//do nothing

		// case packet.TypeDISCONNECT:
	default:
		c.closeConn("invalid packet", true)
	}
}
