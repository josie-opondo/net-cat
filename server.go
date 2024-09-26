package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type Server struct {
	listenAddr string
	ln         net.Listener
	msgChan    chan Message
	clients    map[net.Conn]string // Store client connections with usernames
	sem        chan struct{}
	msgStore   []Message
	shutdown   chan struct{} // Shutdown channel
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
		shutdown:   make(chan struct{}), // Initialize the shutdown channel
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
	go s.handleConnection()

	// Block until a shutdown signal is received
	<-s.shutdown

	// Once shutdown is triggered, close the message channel and all client connections
	close(s.msgChan)        // Close the message channel
	s.closeAllConnections() // Close all active connections

	return nil
}

func (s *Server) handleConnection() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				// Server is shutting down, stop accepting new connections
				return
			default:
				fmt.Printf("Error accepting connection: %v\n", err)
				continue
			}
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
	s.broadcastMsg(conn, []byte(fmt.Sprintf("%s has joined the chat!\r\n", userName)))

	// Send stored messages to the newly connected user
	for _, msg := range s.msgStore {
		conn.Write([]byte(fmt.Sprintf("%s: %s\r\n", msg.sender, string(msg.content))))
	}

	// Read incoming messages from the client
	s.readClientMessages(conn, userName)
}

func (s *Server) readClientMessages(conn net.Conn, username string) {
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			s.broadcastMsg(conn, []byte(fmt.Sprintf("%s has left the chat.\r\n", username)))
			break
		}

		trimmedMsg := strings.TrimSpace(msg)
		formattedMsg := fmt.Sprintf("%s: %s\r\n", username, trimmedMsg)

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
			client.Write(msg) // msg already contains "\r\n"
		}
	}
}

func (s *Server) closeAllConnections() {
	for conn := range s.clients {
		conn.Close() // Close each active client connection
	}
	fmt.Println("All connections closed.")
}

func main() {
	port := ":4080"
	server, err := NewServer(port)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Server running on port:", port)

	// Signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down server...")
		close(server.shutdown) // Trigger shutdown
	}()

	server.Start()
}
