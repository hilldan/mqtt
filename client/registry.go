package client

import (
	"hilldan/mqtt/packet"
	"sync"
)

var (
	TopicFilterRegistry = &topicFilterRegistry{
		subs:   make(map[uint16][]packet.TopicFilter),
		unsubs: make(map[uint16][]packet.String),
	}
	PacketIdRegistry = &packetIdRegistry{
		pids: make(map[uint16]packet.Bit4),
	}
)

type topicFilterRegistry struct {
	sync.RWMutex
	subs   map[uint16][]packet.TopicFilter //packetId->topic filter
	unsubs map[uint16][]packet.String      //packetId->subject
}

func (r *topicFilterRegistry) AddSubs(pid uint16, tfs []packet.TopicFilter) {
	r.Lock()
	r.subs[pid] = tfs
	r.Unlock()
}
func (r *topicFilterRegistry) AddUnsubs(pid uint16, topics []packet.String) {
	r.Lock()
	r.unsubs[pid] = topics
	r.Unlock()
}
func (r *topicFilterRegistry) GetSubs(pid uint16) (tfs []packet.TopicFilter, ok bool) {
	r.RLock()
	tfs, ok = r.subs[pid]
	r.RUnlock()
	return
}
func (r *topicFilterRegistry) GetUnsubs(pid uint16) (ts []packet.String, ok bool) {
	r.RLock()
	ts, ok = r.unsubs[pid]
	r.RUnlock()
	return
}
func (r *topicFilterRegistry) RemoveSubs(pid uint16) {
	r.Lock()
	delete(r.subs, pid)
	r.Unlock()
}
func (r *topicFilterRegistry) RemoveUnsubs(pid uint16) {
	r.Lock()
	delete(r.unsubs, pid)
	r.Unlock()
}

type packetIdRegistry struct {
	sync.RWMutex
	pids map[uint16]packet.Bit4 //latest packetId->control type
}

func (pr *packetIdRegistry) Add(pid packet.Integer, ctype packet.Bit4) {
	pr.Lock()
	pr.pids[uint16(pid)] = ctype
	pr.Unlock()
}
func (pr *packetIdRegistry) Get(pid packet.Integer) (ctype packet.Bit4, ok bool) {
	pr.RLock()
	ctype, ok = pr.pids[uint16(pid)]
	pr.RUnlock()
	return
}
func (pr *packetIdRegistry) Remove(pid packet.Integer) {
	pr.Lock()
	delete(pr.pids, uint16(pid))
	pr.Unlock()
}

// Ignore compare packetId and packet type to decide if client igore the packet with pid.
func (pr *packetIdRegistry) Ignore(pid packet.Integer, ctype packet.Bit4) bool {
	pr.Lock()
	defer pr.Unlock()
	old, ok := pr.pids[uint16(pid)]
	pr.pids[uint16(pid)] = ctype
	return ok && old == ctype
}
