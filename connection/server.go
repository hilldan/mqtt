package connection

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

type Serverer interface {
	Run(handler ServeConn)
}

// ServeConn serve  ervery connect and close the connect after all be done.
type ServeConn func(net.Conn)

// NormalServer is a normal TCP server
type NormalServer struct {
	Addr string
}

func (s *NormalServer) Run(handler ServeConn) {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		ln, err = net.Listen("tcp6", s.Addr)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handler(conn)
	}
}

// TlsServer is a tcp server
type TlsServer struct {
	Addr   string
	Config *tls.Config
}

func (s *TlsServer) Run(handler ServeConn) {
	if s.Config == nil {
		log.Fatal("tls config is nil")
	}

	ln, err := tls.Listen("tcp", s.Addr, s.Config)
	if err != nil {
		ln, err = net.Listen("tcp6", s.Addr)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handler(conn)
	}
}

// WebsocketServer
type WebsocketServer struct {
	Route  string
	Config *websocket.Config
}

func (s *WebsocketServer) Run(handler ServeConn) {
	wserver := websocket.Server{
		Config: *s.Config,
		Handshake: func(config *websocket.Config, req *http.Request) error {
			for _, proto := range config.Protocol {
				if proto == "MQTT" {
					config.Protocol = []string{proto}
					return nil
				}
			}
			return websocket.ErrBadWebSocketProtocol
		},
		Handler: func(cnn *websocket.Conn) {
			handler(cnn)
		},
	}
	http.Handle(s.Route, wserver)
}
