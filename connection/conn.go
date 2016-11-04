package connection

import (
	"log"
	"net"
	"sync"
	"time"
)

type Stat uint8

const (
	StatConn = Stat(iota)
	StatDisconn
)

func (st Stat) String() string {
	switch st {
	case StatConn:
		return "connected"
	case StatDisconn:
		return "disconnected"
	}
	return "invalid status"
}

type Conn struct {
	net.Conn //underlying conn
	sendch   chan []byte
	closech  chan struct{}
	handler  func(data []byte)
	status   Stat   //0 alive, 1 disconnect, 2 ...
	name     string //human readable name
	sync.RWMutex
}

func NewConn(cnn net.Conn) *Conn {
	c := &Conn{
		Conn:    cnn,
		sendch:  make(chan []byte, 1),
		closech: make(chan struct{}),
		name:    cnn.RemoteAddr().String(),
	}
	go c.read()
	go c.write()
	return c
}
func (c *Conn) read() {
	buf := make([]byte, 4096)
	for {
		// 这里不能设置超时，因为需要保持长连接，可能期间并没有通信往来
		//err := c.SetReadDeadline(time.Now().Add(time.Second * 30))
		//当自己主动关闭连接后，再读取会立即返回错误：use of closed network connection
		//当对方连接已经断开后，读取会立即返回n==0，err==io.EOF
		//当对方意外断线，读取会一直堵塞.解决办法是Send with read deadline，超时关闭
		n, err := c.Read(buf)
		if err != nil {
			log.Println(err)
			break
		}
		err = c.SetReadDeadline(time.Time{}) //clear timeout
		if err != nil {
			log.Println(err)
			break
		}
		if c.handler != nil && n > 0 {
			c.handler(buf[:n])
		}
	}
	c.Close()
	c.SetStatus(StatDisconn)
	close(c.closech)
	log.Println("read end")
}
func (c *Conn) write() {
exit:
	for {
		select {
		case <-c.closech:
			break exit
		case msg := <-c.sendch:
			if _, err := c.Write(msg); err != nil {
				log.Println(err)
				c.Close()
				break exit
			}
		}
	}
	log.Println("write end")
}

// Send send data to peer,  will block. for expect the response from client,
// expectReadDeadline is the deadline of conn read, a zero value means no deadline.
func (c *Conn) Send(data []byte, expectReadDeadline time.Time) {
	if c.Status() == StatDisconn {
		return
	}
	err := c.SetReadDeadline(expectReadDeadline)
	if err != nil {
		return
	}
	c.sendch <- data
}

func (c *Conn) SetHandler(h func(data []byte)) {
	c.Lock()
	c.handler = h
	c.Unlock()
}
func (c *Conn) Name() string {
	c.RLock()
	name := c.name
	c.RUnlock()
	return name
}
func (c *Conn) SetName(name string) {
	c.Lock()
	c.name = name
	c.Unlock()
}
func (c *Conn) Status() Stat {
	c.RLock()
	r := c.status
	c.RUnlock()
	return r
}
func (c *Conn) SetStatus(i Stat) {
	c.Lock()
	c.status = i
	c.Unlock()
}
