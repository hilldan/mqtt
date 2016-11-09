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
func (r *topicFilterRegistry) GetRemoveSubs(pid uint16) (tfs []packet.TopicFilter, ok bool) {
	r.Lock()
	defer r.Unlock()
	tfs, ok = r.subs[pid]
	if !ok {
		return
	}
	delete(r.subs, pid)
	return
}
func (r *topicFilterRegistry) GetUnsubs(pid uint16) (ts []packet.String, ok bool) {
	r.RLock()
	ts, ok = r.unsubs[pid]
	r.RUnlock()
	return
}
func (r *topicFilterRegistry) GetRemoveUnsubs(pid uint16) (ts []packet.String, ok bool) {
	r.Lock()
	defer r.Unlock()
	ts, ok = r.unsubs[pid]
	if !ok {
		return
	}
	delete(r.unsubs, pid)
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
