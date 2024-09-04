package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type Server struct {
	listenAddr string
	ln         net.Listener
	netChan    chan struct{}
	msgChan    chan Message
	clients    map[net.Conn]struct{}
	sem        chan struct{}
}

type Message struct {
	sender  string
	content []byte
}

func NewServer(port string) (*Server, error) {
	return &Server{
		listenAddr: port,
		netChan:    make(chan struct{}),
		msgChan:    make(chan Message, 10),
		clients:    make(map[net.Conn]struct{}),
		sem:        make(chan struct{}, 10),
	}, nil
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	go s.handleConnection()

	s.ln = ln
	<-s.netChan

	close(s.msgChan)

	return nil
}

func (s *Server) handleConnection() {
    for {
        conn, err := s.ln.Accept()
        if err != nil {
            fmt.Printf("error accepting connection: %v", err)
            continue
        }

        // Check if the semaphore has space (max 10 connections)
        select {
        case s.sem <- struct{}{}: // Acquire token, proceed if there is space
            go func() {
                defer func() {
                    conn.Close()
                    s.removeClient(conn)
                    <-s.sem // Release token
                }()

                conn.Write([]byte("Hey buddy, what's your name?"))
                userName, _ := bufio.NewReader(conn).ReadString('\n')
                fmt.Println(userName)

                s.addClient(conn)
                s.readConn(conn, strings.TrimSpace(userName))
                log.Printf("received connection: %s", conn.RemoteAddr())
            }()
        default:
            // If the channel is full, reject the connection and print a message
            fmt.Println("Max connections reached, rejecting new connection from:", conn.RemoteAddr())
            conn.Close()
        }
    }
}

func (s *Server) readConn(conn net.Conn, username string) {
	buff := make([]byte, 2048)

	for {
		n, err := conn.Read(buff)
		if err != nil {
			s.broadcastMsg([]byte(fmt.Sprintf("Oops! %s disconnected", username)))
			conn.Close()
			return
		}
		msg := buff[:n]
		s.msgChan <- Message{
			sender:  conn.RemoteAddr().String(),
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

func (s *Server) removeClient(conn net.Conn) {
	delete(s.clients, conn)
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
