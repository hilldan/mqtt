// +build ignore

package main

import (
	"flag"
	"fmt"
	"hilldan/db/redis"
	"hilldan/mqtt"
	"hilldan/mqtt/client"
	"hilldan/mqtt/connection"
	"hilldan/mqtt/packet"
	"log"
	"time"
)

var (
	cid = flag.String("cid", "client_*", "client id")
)

type listener struct {
	mqtt.DefaultListener
}

func (e listener) OnPublishReceived(p packet.PublishPacket) {
	fmt.Println("received:", p.Qos, p.Dup, p.ApplicationMessage)
}
func (e listener) OnSubscribeSuccess(tfs []packet.TopicFilter) {
	fmt.Println("subscribe:", tfs)
}

func main() {
	flag.Parse()
	c := &connection.NormalClient{
		Network: "tcp",
		Addr:    "127.0.0.1:2000",
	}
	persister := mqtt.NewRedisPersist(redis.NewClient("127.0.0.1:6379", "", 0, 10))
	cnnPacket := &packet.ConnectPacket{
		UserNameFlag: true,
		PasswdFlag:   true,
		WillRetain:   false,
		WillQoS:      0,
		WillFlag:     false,
		CleanSession: false,
		KeepAlive:    10,
		ClientId:     packet.String(*cid),
		WillTopic:    "",
		WillMessage:  "",
		UserName:     "xxx",
		Password:     "yyy",
	}

	client.SetEventListener(listener{})
	cnn, err := client.RunMQTT(c, persister, cnnPacket)
	if err != nil {
		log.Println(err)
		return
	}
	defer cnn.Disconnect()

	// filters := []packet.TopicFilter{
	// 	{Topic: "a/b/c", Qos: 1},
	// 	{Topic: "f/#"},
	// }
	// cnn.Subscribe(filters)
	// time.Sleep(10e9)
	pub := packet.PublishPacket{
		Qos:                2,
		Retain:             false,
		TopicName:          "f/b/c",
		ApplicationMessage: "hello world ########",
	}
	cnn.Publish(pub)
	cnn.Publish(pub)
	cnn.Publish(pub)
	cnn.Publish(pub)
	cnn.Publish(pub)

	// unsubs := []packet.String{
	// 	"a/b/c",
	// }
	// cnn.Unsubscribe(unsubs)

	time.Sleep(30e9)

}
