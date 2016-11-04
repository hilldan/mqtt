package server

import (
	"encoding/json"
	"errors"
	"hilldan/mqtt/packet"
	"log"
	"sync"
)

const (
	KeyRetain  = "mq:sr"
	KeySession = "mq:ss"
)

var (
	ConnRegistry = &connRegistry{
		Conns: make(map[string]*mqttConn),
	}
	RetainRegistry   *retainRegistry
	WildcardRegistry = &wildcardRegistry{
		path: make(map[string][]string),
	}
	PacketIdRegistry = &packetIdRegistry{
		pids: make(map[string]packet.Integer),
	}
)

type connRegistry struct {
	Conns map[string]*mqttConn //clientId->conn
	sync.RWMutex
}

func (cr *connRegistry) Add(key string, c *mqttConn) {
	cr.Lock()
	if old, ok := cr.Conns[key]; ok {
		old.closeConn("old conn", true)
		// PacketIdRegistry.Remove(key)
		delete(cr.Conns, key)
	}
	cr.Conns[key] = c
	cr.Unlock()
}

func (cr *connRegistry) Remove(key string) {
	cr.Lock()
	delete(cr.Conns, key)
	cr.Unlock()
}

func (cr *connRegistry) Get(key string) (c *mqttConn, ok bool) {
	cr.RLock()
	defer cr.RUnlock()
	c, ok = cr.Conns[key]
	return
}

func (cr *connRegistry) Publish(p packet.PublishPacket, excludeId string) {
	cr.Lock()
	defer cr.Unlock()
	for _, c := range cr.Conns {
		if string(c.clientId) == excludeId {
			continue
		}
		if matched, max := match(c.session.GetSubscription(), string(p.TopicName)); matched {
			if p.Qos > max {
				p.Qos = max
			}
			go c.publish(p)
		}
	}
}

// retainRegistry manages publish packet retained, such as persistance, delete
type retainRegistry struct {
	sync.RWMutex
	PubRetain map[string]packet.PublishPacket //topic->packet
}

func NewRetainRegistry() *retainRegistry {
	datas, err := persister.LoadAll(KeyRetain)
	if err != nil {
		log.Printf("load all retained packet err: %v", err)
	}

	m := make(map[string]packet.PublishPacket)
	for k, v := range datas {
		p := new(packet.PublishPacket)
		err = json.Unmarshal(v, p)
		if err == nil {
			m[k] = *p
		}
	}

	return &retainRegistry{
		PubRetain: m,
	}
}

func (rg *retainRegistry) Add(topic string, p packet.PublishPacket) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	err = persister.Save(KeyRetain, topic, data)
	if err != nil {
		return err
	}

	rg.Lock()
	rg.PubRetain[topic] = p
	rg.Unlock()
	return nil
}
func (rg *retainRegistry) Get(topic string) (p packet.PublishPacket, ok bool) {
	rg.RLock()
	p, ok = rg.PubRetain[topic]
	rg.RUnlock()
	return
}
func (rg *retainRegistry) Remove(topic string) {
	err := persister.Delete(KeyRetain, topic)
	if err != nil {
		return
	}
	rg.Lock()
	delete(rg.PubRetain, topic)
	rg.Unlock()
}

func (rg *retainRegistry) Publish(sub packet.TopicFilter, c *mqttConn) {
	rg.RLock()
	defer rg.RUnlock()
	for k, v := range rg.PubRetain {
		if matchOne(string(sub.Topic), k) {
			if v.Qos > sub.Qos {
				v.Qos = sub.Qos
			}
			go c.publish(v)
		}
	}
}

type wildcardRegistry struct {
	sync.RWMutex
	path map[string][]string
}

func (wr *wildcardRegistry) Get(sub string) (subs []string, err error) {
	wr.Lock()
	defer wr.Unlock()
	subs, ok := wr.path[sub]
	if ok {
		return
	}
	ok, subs = check(sub)
	if !ok {
		err = errors.New("invalid subject: " + sub)
		return
	}
	wr.path[sub] = subs
	return
}
func (wr *wildcardRegistry) Remove(sub string) {
	wr.Lock()
	delete(wr.path, sub)
	wr.Unlock()
}

type packetIdRegistry struct {
	sync.RWMutex
	pids map[string]packet.Integer //clientId->latest packetId
}

func (pr *packetIdRegistry) Add(clientId string, pid packet.Integer) {
	pr.Lock()
	pr.pids[clientId] = pid
	pr.Unlock()
}
func (pr *packetIdRegistry) Get(clientId string) (pid packet.Integer, ok bool) {
	pr.RLock()
	pid, ok = pr.pids[clientId]
	pr.RUnlock()
	return
}
func (pr *packetIdRegistry) Remove(clientId string) {
	pr.Lock()
	delete(pr.pids, clientId)
	pr.Unlock()
}

// Ignore compare packetIds to decide if server igore the packet with pid.
func (pr *packetIdRegistry) Ignore(clientId string, pid packet.Integer) bool {
	pr.Lock()
	defer pr.Unlock()
	old, ok := pr.pids[clientId]
	pr.pids[clientId] = pid
	return ok && old == pid
}
