package mqtt

import (
	"errors"

	"hilldan/mqtt/packet"
)

var (
	ErrTimeout = errors.New("Timeout")
	ErrConnect = errors.New("Connect refused")
	ErrAuth    = errors.New("Auth fail")
)

type PacketReaded struct {
	P   packet.ControlPacketer
	Err error
}
