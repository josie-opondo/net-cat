package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type Server struct {
	listenAddr string
	ln         net.Listener
	msgChan    chan Message
	clients    map[net.Conn]string
	sem        chan struct{}
	msgStore   []Message
}

type Message struct {
	sender  string
	content []byte
	conn    net.Conn
}

func NewServer(port string) (*Server, error) {
	return &Server{
		listenAddr: port,
		msgChan:    make(chan Message, 10),
		clients:    make(map[net.Conn]string),
		sem:        make(chan struct{}, 10),
		msgStore:   make([]Message, 0),
	}, nil
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	s.ln = ln

	go func() {
		for msg := range s.msgChan {
			s.broadcastMsg(msg.conn, msg.content)
		}
	}()

	s.handleConnection()
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
			go s.handleClient(conn)
		default:
			fmt.Println("Max connections reached, rejecting new connection from:", conn.RemoteAddr())
			conn.Close()
		}
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		s.removeClient(conn)
		conn.Close()
		<-s.sem // Release token
	}()

	conn.Write([]byte("Hey buddy, what's your name? "))
	userName, _ := bufio.NewReader(conn).ReadString('\n')
	userName = strings.TrimSpace(userName)
	if userName == "" {
		conn.Write([]byte("Username cannot be empty. Disconnecting...\n"))
		return
	}

	s.addClient(conn, userName)

	s.broadcastMsg(conn, []byte(fmt.Sprintf("%s has joined the chat!\n", userName)))

	for _, mess := range s.msgStore {
		conn.Write([]byte(fmt.Sprintf("%s: %s\n", mess.sender, mess.content)))
	}

	s.readClientMessages(conn, userName)
}

func (s *Server) readClientMessages(conn net.Conn, username string) {
	reader := bufio.NewReader(conn)

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			s.broadcastMsg(conn, []byte(fmt.Sprintf("\nOops! %s disconnected\n", username)))
			break
		}

		formatMsg := []byte(fmt.Sprintf("\n%s: %s\n", username, strings.TrimSpace(string(msg))))

		message := Message{
			sender:  username,
			content: formatMsg,
			conn:    conn,
		}

		s.msgStore = append(s.msgStore, message)
		s.msgChan <- message
	}
}

func (s *Server) addClient(conn net.Conn, userName string) {
	s.clients[conn] = userName
}

func (s *Server) broadcastMsg(sender net.Conn, msg []byte) {
	for client := range s.clients {
		if client != sender {
			client.Write(msg)
		}
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
		return
	}
	fmt.Println("Server running on port: ", port)

	server.Start()
}
