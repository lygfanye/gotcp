package main

import (
	"fmt"
	"log"
	"net"

	"github.com/lygfanye/gotcp/examples/echo"
	"time"
	"sync"

)

func main() {

	retry :=0
	for {

		if retry > 10 {
			fmt.Println("重连失败")
			break
		}else if retry > 0 {
			fmt.Println("第", retry, "次重连")
		}

		tcpAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:8980")
		if err !=nil {
			retry++
			continue
		}


		conn, err := net.DialTimeout("tcp", tcpAddr.String(), 100 * time.Second)
		tcpConn,_:=conn.(*net.TCPConn)
		if err !=nil {
			fmt.Println("Srv is lose, retry dial .....")
			retry++
			time.Sleep(10 * time.Second)
			continue
		}

		retry = 0
		echoProtocol := &echo.EchoProtocol{}
		stopCh := make(chan struct{})

		var wg *sync.WaitGroup
		wg = &sync.WaitGroup{}
		go func() {
			wg.Add(1)
			defer func() {
				wg.Done()
			}()
			for{
				select {
				case <-stopCh:
					fmt.Println("write exit")
					return
				default:
					tcpConn.Write(echo.NewEchoPacket([]byte("hello"), false).Pack())
					log.Println("Write to server")
					time.Sleep(2 * time.Second)
				}
			}
		}()

		for {

			p, err := echoProtocol.ReadPacket(tcpConn)
			if err == nil {
				echoPacket := p.(*echo.EchoPacket)
				fmt.Printf("Server reply:[%v] [%v]\n", echoPacket.GetLength(), string(echoPacket.GetBody()))
			}else {
				break
			}
		}

		close(stopCh)
		tcpConn.Close()
		wg.Wait()
		fmt.Println("Srv is lose .....")
		retry ++
	}

}
