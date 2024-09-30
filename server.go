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
	clients    map[net.Conn]string
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

	go func() {
		for msg := range s.msgChan {
			s.broadcastMsg(msg.conn, msg.content)
		}
	}()

	go s.handleConnection()

	<-s.shutdown

	close(s.msgChan)
	s.closeAllConnections()

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

	if userName == "" {
		conn.Write([]byte("Username cannot be empty. Disconnecting...\n"))
		return
	}

	s.addClient(conn, userName)

	s.broadcastMsg(conn, []byte(fmt.Sprintf("%s has joined the chat!", userName)))

	for _, msg := range s.msgStore {
		conn.Write([]byte(fmt.Sprintf("%s: %s\r\n", msg.sender, string(msg.content))))
	}

	s.readConn(conn, strings.TrimSpace(userName))
}

func (s *Server) readConn(conn net.Conn, username string) {
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
			content: []byte(formatMsg),
			conn:    conn,
		}

		// Store and broadcast the message
		s.msgStore = append(s.msgStore, message)
		s.msgChan <- message
	}
}


func (s *Server) addClient(conn net.Conn, userName string) {
	s.clients[conn] = userName
}

func (s *Server) broadcastMsg(conn net.Conn, msg []byte) {
	for client := range s.clients {
		if client != conn {
			client.Write(msg)
		}
	}
}

func (s *Server) removeClient(conn net.Conn) {
	delete(s.clients, conn)
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
	fmt.Println("Server running on port: ", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nServer shutting down...")
		close(server.shutdown)
	}()
	server.Start()
}
