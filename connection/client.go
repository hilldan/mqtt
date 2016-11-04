package connection

import (
	"crypto/tls"
	"net"

	"golang.org/x/net/websocket"
)

type Clienter interface {
	Dial() (net.Conn, error)
}
type NormalClient struct {
	Dialer        *net.Dialer
	Network, Addr string
}

func (c *NormalClient) Dial() (cnn net.Conn, err error) {
	if c.Dialer != nil {
		cnn, err = c.Dialer.Dial(c.Network, c.Addr)
	} else {
		cnn, err = net.Dial(c.Network, c.Addr)
	}
	return
}

type TlsClient struct {
	Config        *tls.Config
	Dialer        *net.Dialer
	Network, Addr string
}

func (c *TlsClient) Dial() (cnn net.Conn, err error) {
	if c.Dialer != nil {
		cnn, err = tls.DialWithDialer(c.Dialer, c.Network, c.Addr, c.Config)
	} else {
		cnn, err = tls.Dial(c.Network, c.Addr, c.Config)
	}
	return
}

type WebsocketClient struct {
	UrlAddress string
	UrlOrigin  string
	Config     *websocket.Config
}

func (c *WebsocketClient) Dial() (cnn net.Conn, err error) {
	if c.Config != nil {
		c.Config.Protocol = []string{"MQTT"}
		cnn, err = websocket.DialConfig(c.Config)
	} else {
		cnn, err = websocket.Dial(c.UrlAddress, "MQTT", c.UrlOrigin)
	}
	return
}
