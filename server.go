package main

import (
	"fmt"
	"log"
	"net"
)

type Server struct {
	listenAddr string
	ln net.Listener
	netChan chan struct{}
	msgChan chan Message
	clients map[net.Conn]struct{}
}

type Message struct {
	sender string
	content []byte
}

func NewServer(port string) (*Server, error) {
	return &Server{
		listenAddr: port,
		netChan: make(chan struct{}),
		msgChan: make(chan Message, 10),
		clients: make(map[net.Conn]struct{}),
	}, nil
}

func (s *Server) Start() error{
	ln, err := net.Listen("tcp", s.listenAddr)
	if err!= nil {
        return err
    }
	defer ln.Close()

	go s.handleConnetion()

	s.ln = ln
	<-s.netChan

	close(s.msgChan)

	return nil
}

func (s *Server) handleConnetion() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Printf("error accepting connection: %v", err)
			continue
		}
		s.addClient(conn)
		go s.readConn(conn)
		log.Printf("received connection: %s", conn.RemoteAddr())
	}
}

func (s *Server) readConn(conn net.Conn) {
	buff := make([]byte, 2048)

	for {
		n, err := conn.Read(buff)
		if err != nil {
			fmt.Println("Oops! He disconnected")
            conn.Close()
            return
		}
		msg := buff[:n]
		s.msgChan <- Message{
			sender: conn.RemoteAddr().String(),
            content: msg,
		}
		conn.Write([]byte(" 👉 :"))
	}
}

func (s *Server) addClient(conn net.Conn) {
	s.clients[conn] = struct{}{}
}

func (s *Server) broadcastMsg(msg []byte) {
	for client := range s.clients {
		client.Write(msg)
	}
}
func main() {
	port := ":8080"
	server, err := NewServer(port)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Server running on port: ", port)
	go func() {
        for msg := range server.msgChan {
            server.broadcastMsg(msg.content)
        }
    }()
	server.Start()
}