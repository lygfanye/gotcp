package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/lygfanye/gotcp"
	"github.com/lygfanye/gotcp/examples/telnet"
)

func AfterConnect(c *gotcp.Conn) {
	addr := c.GetRawConn().RemoteAddr()
	c.PutExtraData(addr)
	fmt.Println("OnConnect:", addr)
}

func main() {

	// creates a tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":23")
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err)

	sc := &gotcp.ServerConfig{
		SendChanBuf:    6,
		ReceiveChanBuf: 6,
		AfterConnect:   AfterConnect,
		AfterReceive:   nil,
		AfterClose:     nil,
	}

	srv := gotcp.NewServer(sc, &telnet.TelnetProtocol{})

	// starts service
	go srv.Start(listener)
	fmt.Println("listening:", listener.Addr())

	// catchs system signal
	chSig := make(chan os.Signal)
	signal.Notify(chSig, syscall.SIGTERM)
	fmt.Println("Signal: ", <-chSig)

	// stops service
	srv.Stop(listener)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
