package mqtt

import "hilldan/mqtt/packet"

// EventListener
type EventListener interface {
	// OnConnected be called after the connection establisded.
	// if the result error is not nil, connection will be closed
	OnConnected(p packet.ConnectPacket) error
	OnPublishReceived(p packet.PublishPacket)
	OnSubscribeSuccess(tfs []packet.TopicFilter)
	OnUnsubscribeSuccess(tfs []packet.String)
	OnDisconnected()
}

type DefaultListener struct{}

func (e DefaultListener) OnConnected(p packet.ConnectPacket) error    { return nil }
func (e DefaultListener) OnPublishReceived(p packet.PublishPacket)    {}
func (e DefaultListener) OnSubscribeSuccess(tfs []packet.TopicFilter) {}
func (e DefaultListener) OnUnsubscribeSuccess(tfs []packet.String)    {}
func (e DefaultListener) OnDisconnected()                             {}
