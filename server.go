package gotcp

import (
	"sync"
	"log"
	"net"
)

type ServerConfig struct {
	EnableHeartBeating    bool  //启用心跳检测
	HeartBeatingThreshold int64 //心跳过滤阀值
	HeartBeatingPeriod    int64 //心跳检查周期
	//EnableTcpKeepAlive bool //启用tcp keepalive机制

	SendChanBuf    int
	ReceiveChanBuf int

	AfterConnect func(*Conn)
	AfterReceive func(*Conn, Packet)
	AfterSend    func(*Conn, Packet)
	AfterClose   func(*Conn)
}

type Server struct {
	config   *ServerConfig
	connSet  *sync.Map
	wg       *sync.WaitGroup
	protocol Protocol
	stopChan chan struct{}
}

func NewServer(config *ServerConfig, protocol Protocol) *Server {
	s := &Server{
		connSet:  &sync.Map{},
		config:   config,
		wg:       &sync.WaitGroup{},
		stopChan: make(chan struct{}),
		protocol: protocol,
	}

	return s
}

func (s *Server) Conn(id string) (*Conn, bool) {
	if v, ok := s.connSet.Load(id); ok {
		return v.(*Conn), ok
	}

	return nil, false
}

func (s *Server) StoreConn(id string, conn *Conn) {
	s.connSet.Store(id, conn)
}

func (s *Server) RemoveConn(id string) {
	s.connSet.Delete(id)
}

func (s *Server) ConnSize() (int) {
	var counter int

	s.connSet.Range(func(k, v interface{}) bool {
		counter++
		return true
	})

	return counter
}

func (s *Server) Start(l *net.TCPListener) {
	s.wg.Add(1)
	defer func() {
		s.wg.Done()
	}()

	for {
		select {
		case <-s.stopChan:
			return
		default:
		}


		connection, err := l.AcceptTCP()
		if err != nil {
			continue
		}

		s.wg.Add(1)
		go func() {
			newConn(connection, s).Do()
			s.wg.Done()
		}()
	}

}

func (s *Server) Stop(l *net.TCPListener) {
	close(s.stopChan)
	log.Println("wati conn process done....")
	l.Close()
	s.wg.Wait()
	log.Println("server is stopped....")
}

func CheckError(str string, err error) {
	if err != nil {
		log.Println("Error, ", str, ", ", err)
		panic(err)
	}
}
