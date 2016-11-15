// +build ignore

package main

import (
	"fmt"
	"hilldan/db/redis"
	"hilldan/mqtt"
	"hilldan/mqtt/connection"
	"hilldan/mqtt/packet"
	"hilldan/mqtt/server"
)

type listener struct {
	mqtt.DefaultListener
}

func (e listener) OnPublishReceived(p packet.PublishPacket) {
	fmt.Println("received:", p.PacketId)
}
func (e listener) OnSubscribeSuccess(tfs []packet.TopicFilter) {
	fmt.Println("subscribe:", tfs)
}

func main() {
	s := &connection.NormalServer{Addr: "127.0.0.1:2000"}
	p := mqtt.NewRedisPersist(redis.NewClient("127.0.0.1:6379", "", 0, 10))
	auth := func(user, passwd string) bool { return true }
	server.SetAuthFunc(auth)
	server.SetEventListener(listener{})
	server.RunMQTT(s, p)
}
