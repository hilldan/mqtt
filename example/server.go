package main

import (
	"hilldan/db/redis"
	"hilldan/mqtt"
	"hilldan/mqtt/connection"
	"hilldan/mqtt/server"
)

func main() {
	s := &connection.NormalServer{Addr: "127.0.0.1:2000"}
	p := mqtt.NewRedisPersist(redis.NewClient("127.0.0.1:6379", "", 0, 10))
	auth := func(user, passwd string) bool { return true }
	server.RunMQTT(s, p, auth)
}
