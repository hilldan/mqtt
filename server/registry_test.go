package server

import (
	"fmt"
	"hilldan/db/redis"
	"hilldan/mqtt"
	"hilldan/mqtt/packet"
	"sync"
	"testing"
)

func initPersister() {
	db := redis.NewClient("127.0.0.1:6379", "", 0, 10)
	SetPersister(mqtt.NewRedisPersist(db))
}

func TestPersistRetain(t *testing.T) {
	initPersister()
	r := NewRetainRegistry()
	o := packet.PublishPacket{
		Dup:                true,
		Qos:                1,
		Retain:             false,
		TopicName:          "topicName",
		PacketId:           134,
		ApplicationMessage: "message",
	}
	var wg sync.WaitGroup
	var err error
	const N = 100000
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			topic := fmt.Sprintf("%d", i)
			o.TopicName = packet.String(topic)
			err = r.Add(topic, o)
			if err != nil {
				t.Errorf("add retain packet err : %v", err)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	rr := NewRetainRegistry()
	for k, v := range rr.PubRetain {
		if vv, ok := r.PubRetain[k]; !ok || v != vv {
			t.Errorf("load all err")
		}
	}
	if len(rr.PubRetain) != N {
		t.Errorf("load all err,want %d actual %d", N, len(rr.PubRetain))
	}
}

func TestPersistSession(t *testing.T) {
	// initPersister()
	//
}
