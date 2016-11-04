package mqtt

import (
	"encoding/json"
	"sync"

	"hilldan/mqtt/packet"
)

// Session:
// A stateful interaction between a Client and a Server.
// Some Sessions last only as long as the Network Connection,
// others can span multiple consecutive Network Connections between a Client and a Server.
type Session struct {
	sync.RWMutex
	//have not been completely acknowledged
	PubOut map[uint16]packet.PublishPacket //packetId->packet

	//The Client’s subscriptions.
	Subscript []packet.TopicFilter
}

func (s *Session) AddPubOut(packetId packet.Integer, p packet.PublishPacket) {
	s.Lock()
	s.PubOut[uint16(packetId)] = p
	s.Unlock()
}
func (s *Session) GetPubOut(packetId packet.Integer) (p packet.PublishPacket, ok bool) {
	s.RLock()
	p, ok = s.PubOut[uint16(packetId)]
	s.RUnlock()
	return
}
func (s *Session) RemovePubOut(packetId packet.Integer) {
	s.Lock()
	delete(s.PubOut, uint16(packetId))
	s.Unlock()
}
func (s *Session) SetSubscription(sub []packet.TopicFilter) {
	s.Lock()
	s.Subscript = sub
	s.Unlock()
}
func (s *Session) GetSubscription() []packet.TopicFilter {
	s.RLock()
	b := make([]packet.TopicFilter, len(s.Subscript))
	copy(b, s.Subscript)
	s.RUnlock()
	return b
}
func (s *Session) AddSubscription(tfs []packet.TopicFilter) {
	s.Lock()
	s.Subscript = append(s.Subscript, tfs...)
	s.Unlock()
}
func (s *Session) Unsubscription(unsub []packet.String) {
	if len(unsub) == 0 {
		return
	}
	s.Lock()
	defer s.Unlock()
	for _, v := range unsub {
		for k, vv := range s.Subscript {
			if v == vv.Topic {
				s.Subscript = append(s.Subscript[:k], s.Subscript[k+1:]...)
			}
		}
	}
}

func NewSession() *Session {
	return &Session{
		PubOut:    make(map[uint16]packet.PublishPacket),
		Subscript: make([]packet.TopicFilter, 0),
	}
}

func (s *Session) Save(key, clientId string, persister Persister) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return persister.Save(key, clientId, b)
}
