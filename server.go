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
	clients    map[net.Conn]string // Store client connections with usernames
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

	// Goroutine to handle incoming messages and broadcast them
	go func() {
		for msg := range s.msgChan {
			s.broadcastMsg(msg.conn, msg.content)
		}
	}()

	// Handle incoming connections
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

		// Semaphore to limit connections
		select {
		case s.sem <- struct{}{}: // Allow connection
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
		<-s.sem // Release the semaphore
	}()

	conn.Write([]byte("Hey buddy, what's your name?\n"))
	userName, _ := bufio.NewReader(conn).ReadString('\n')
	userName = strings.TrimSpace(userName)

	if userName == "" {
		conn.Write([]byte("Invalid name. Disconnecting.\n"))
		return
	}

	s.addClient(conn, userName)

	// Broadcast that the user has joined
	s.broadcastMsg(conn, []byte(fmt.Sprintf("%s has joined the chat!\n", userName)))

	// Send stored messages to the newly connected user
	for _, msg := range s.msgStore {
		conn.Write([]byte(fmt.Sprintf("%s: %s\n", msg.sender, string(msg.content))))
	}

	// Read incoming messages from the client
	s.readClientMessages(conn, userName)
}

func (s *Server) readClientMessages(conn net.Conn, username string) {
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			s.broadcastMsg(conn, []byte(fmt.Sprintf("%s has left the chat.\n", username)))
			break
		}

		trimmedMsg := strings.TrimSpace(msg)
		formattedMsg := fmt.Sprintf("%s: %s", username, trimmedMsg)

		message := Message{
			sender:  username,
			content: []byte(formattedMsg),
			conn:    conn,
		}

		// Store and broadcast the message
		s.msgStore = append(s.msgStore, message)
		s.msgChan <- message
	}
}

func (s *Server) addClient(conn net.Conn, username string) {
	s.clients[conn] = username
}

func (s *Server) removeClient(conn net.Conn) {
	delete(s.clients, conn)
}

func (s *Server) broadcastMsg(sender net.Conn, msg []byte) {
	for client := range s.clients {
		if client != sender { // Don't send the message back to the sender
			client.Write(msg)
		}
	}
}

func main() {
	port := ":8080"
	server, err := NewServer(port)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Server running on port:", port)
	server.Start()
}
