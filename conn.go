package gotcp

import (
	"errors"
	"net"
	"sync"
	"time"
	"fmt"
	"log"
)

// Error type
var (
	ErrConnClosing   = errors.New("use of closed network connection")
	ErrWriteBlocking = errors.New("write packet was blocking")
	ErrWriteTimeout  = errors.New("write packet timeout")
	ErrReadBlocking  = errors.New("read packet was blocking")
)

type Conn struct {
	id          string
	srv         *Server
	extraData   interface{}
	closeOnce   sync.Once
	rawConn     *net.TCPConn
	receiveChan chan Packet
	sendChan    chan Packet
	heart       int64
	closeChan   chan struct{}
}

func newConn(rawConn *net.TCPConn, srv *Server) *Conn {
	return &Conn{
		srv:         srv,
		rawConn:     rawConn,
		receiveChan: make(chan Packet, srv.config.ReceiveChanBuf),
		sendChan:    make(chan Packet, srv.config.SendChanBuf),
		closeChan:   make(chan struct{}),
		heart:       time.Now().Unix(),
	}
}

// GetExtraData gets the extra data from the Conn
func (c *Conn) GetExtraData() interface{} {
	return c.extraData
}

// PutExtraData puts the extra data with the Conn
func (c *Conn) PutExtraData(data interface{}) {
	c.extraData = data
}

func (c *Conn) GetRawConn() *net.TCPConn {
	return c.rawConn
}

func (c *Conn) SetHeartBeat(heart int64) {
	c.heart = heart
}

func (c *Conn) GetHeartBeat() int64 {
	return c.heart
}

func (c *Conn) AsyncWritePacket(p Packet, timeout time.Duration) error {
	if timeout == 0 {
		select {
		case c.sendChan <- p:
			return nil
		default:
			return ErrWriteBlocking
		}
	} else {
		select {
		case c.sendChan <- p:
			return nil
		case <-time.After(timeout):
			return ErrWriteTimeout
		case <-c.closeChan:
			return ErrConnClosing
		}
	}
}

func (c *Conn) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("ReadLoop Error", err)
		}
		c.Close()
	}()

	for {
		select {
		case <-c.srv.stopChan:
			log.Println("Tcp server is stop, ReadLoop exit")
			return
		case <-c.closeChan:
			log.Println("client conn is closed, ReadLoop exit")
			return
		default:
		}

		c.SetHeartBeat(time.Now().Unix())
		p, err := c.srv.protocol.ReadPacket(c.rawConn)
		if err != nil {
			log.Println("client conn is closed, ReadLoop exit")
			return
		}
		c.receiveChan <- p

	}
}

func (c *Conn) writeLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("WriteLoop Error", err)
		}
		c.Close()
	}()

	for {
		select {
		case <-c.srv.stopChan:
			log.Println("Tcp server is stop, WriteLoop exit")
			return
		case <-c.closeChan:
			log.Println("client conn is closed, WriteLoop exit")
			return
		case p, ok := <-c.sendChan:
			if ok {
				if _, err := c.rawConn.Write(p.Pack()); err != nil {
					log.Println("cloud not send packet to client, HandLoop exit")
					return
				}

				if c.srv.config.AfterSend != nil {
					c.srv.config.AfterSend(c, p)
				}
			}

		default:
		}
	}
}

func (c *Conn) handleLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("HandleLoop Error", err)
		}
		c.Close()
	}()

	for {
		select {
		case <-c.srv.stopChan:
			log.Println("Tcp server is stop, HandLoop exit")
			return
		case <-c.closeChan:
			log.Println("client conn is closed, HandLoop exit")
			return
		case p, ok := <-c.receiveChan:
			if ok {
				log.Println("Receive Client Pakcet")
				if c.srv.config.AfterReceive != nil {
					c.srv.config.AfterReceive(c, p)
				}
			}
		}

	}
}

func (c *Conn) Do() {
	fmt.Println("Conn begin process.....")

	if c.srv.config.AfterConnect != nil {
		c.srv.config.AfterConnect(c)
	}

	asyncDo(c.handleLoop, c.srv.wg)
	asyncDo(c.readLoop, c.srv.wg)
	asyncDo(c.writeLoop, c.srv.wg)

	if c.srv.config.EnableHeartBeating {
		asyncDo(c.checkHeart, c.srv.wg)
	}
}

func (c *Conn) checkHeart() {
	tick := time.NewTicker(time.Duration(c.srv.config.HeartBeatingPeriod) * time.Second)
	defer func() {
		tick.Stop()
	}()
	for {
		select {
		case <-c.srv.stopChan:
			log.Println("Tcp server is stop, checkHeart exit")
			return
		case <-c.closeChan:
			log.Println("client conn is closed, checkHeart exit")
			return
		case <-tick.C:
			diff := time.Now().Unix() - c.GetHeartBeat()
			if diff > c.srv.config.HeartBeatingThreshold {
				log.Println(c.rawConn.RemoteAddr(), " timeout close")
				c.Close()
				return
			}else {
				log.Println(c.rawConn.RemoteAddr(), " expired at ", c.heart ,time.Unix(c.heart + c.srv.config.HeartBeatingThreshold, 0).Format("2006-01-02 15:04:05"))
			}
		}
	}
}

func (c *Conn) Close() {
	c.closeOnce.Do(func() {
		close(c.receiveChan)
		close(c.sendChan)
		close(c.closeChan)
		c.srv.RemoveConn(c.id)
		c.rawConn.Close()
		if c.srv.config.AfterClose != nil {
			c.srv.config.AfterClose(c)
		}
	})
}

func asyncDo(fn func(), wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		fn()
		wg.Done()
	}()
}
