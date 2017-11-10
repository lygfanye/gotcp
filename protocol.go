package gotcp

import (
	"net"
)

type Packet interface {
	Pack() []byte
}

type Protocol interface {
	ReadPacket(conn *net.TCPConn) (Packet, error)
}
