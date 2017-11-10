package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/lygfanye/gotcp"
	"github.com/lygfanye/gotcp/examples/echo"
)

func AfterConnect(c *gotcp.Conn) {
	addr := c.GetRawConn().RemoteAddr()
	c.PutExtraData(addr)
	fmt.Println("OnConnect:", addr)
}

func AfterReceive(c *gotcp.Conn, p gotcp.Packet) {
	echoPacket := p.(*echo.EchoPacket)
	fmt.Printf("OnMessage:[%v] [%v]\n", echoPacket.GetLength(), string(echoPacket.GetBody()))
	c.AsyncWritePacket(echo.NewEchoPacket(echoPacket.Pack(), true), time.Second)
}

func AfterClose(c *gotcp.Conn) {
	fmt.Println("OnClose:", c.GetExtraData())
}

func main() {

	sc := &gotcp.ServerConfig{
		SendChanBuf:    6,
		ReceiveChanBuf: 6,
		AfterConnect:   AfterConnect,
		AfterReceive:   AfterReceive,
		AfterClose:     AfterClose,
	}

	// creates a tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":8980")
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	srv := gotcp.NewServer(sc, &echo.EchoProtocol{})

	// starts service
	go srv.Start(listener)
	fmt.Println("listening:", listener.Addr())

	// catchs system signal
	chSig := make(chan os.Signal)
	signal.Notify(chSig, os.Interrupt)
	<-chSig
	// stops service
	srv.Stop(listener)

}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
