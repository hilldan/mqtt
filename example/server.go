package main

import (
	"errors"
	"hilldan/db/redis"
	"hilldan/mqtt"
	"hilldan/mqtt/connection"
	"hilldan/mqtt/packet"
	"hilldan/mqtt/server"
)

func main() {
	s := &connection.NormalServer{Addr: "127.0.0.1:2000"}
	p := mqtt.NewRedisPersist(redis.NewClient("127.0.0.1:6379", "", 0, 10))
	auth := func(user, passwd string) bool { return true }
	f := func(p *packet.ConnectPacket) error {
		pub := packet.PublishPacket{
			TopicName:          "xx/uu",
			ApplicationMessage: "xxxxxxxxx",
		}
		server.Publish(pub, string(p.ClientId))
		return errors.New("finish")
	}
	server.SetAuthFunc(auth)
	server.SetDoWhenConnect(f)
	server.RunMQTT(s, p)
}
