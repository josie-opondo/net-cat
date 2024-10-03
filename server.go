package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
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
	msgDate time.Time
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

func (s *Server) Logo() (string, error) {
	logo, err := os.ReadFile("hello.txt")
	if err != nil {
		return "", err
	}

	return string(logo), nil
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
			conn.Write([]byte("Oops, the chatroom is at max capacity. Try again later..."))
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

	logo, _ := s.Logo()

	welcomeMessage := fmt.Sprintf("Welcome to TCP-Chat!\n%s\n[ENTER YOUR NAME]: ", logo)

	conn.Write([]byte(welcomeMessage))
	userName, _ := bufio.NewReader(conn).ReadString('\n')

	if len(strings.TrimSpace(userName)) < 3 {
		conn.Write([]byte("Enter valid name. Disconnecting...\n"))
		return
	}

	s.addClient(conn, userName)
	conn.Write([]byte(fmt.Sprintf("Welcome, %s!\n", userName[:len(userName)-1])))

	msg := fmt.Sprintf("%s has joined the chat!\n", userName[:len(userName)-1])

	s.clientInfomer(conn, []byte(msg))

	for _, msg := range s.msgStore {
		timestamp := msg.msgDate.Format("2006-01-02 15:04:05")
		message := fmt.Sprintf("[%v][%s]:%s\n", timestamp, msg.sender, string(msg.content))
		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Error writing to connection:", err)
		}
	}

	s.readConn(conn, strings.TrimSpace(userName))
}

func (s *Server) readConn(conn net.Conn, username string) {
	reader := bufio.NewReader(conn)

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			s.clientInfomer(conn, []byte(fmt.Sprintf("\nOops! %s disconnected\n", s.clients[conn])))
			break
		}

		formatMsg := s.handleUserInput(conn, msg)
		if formatMsg == nil {
			continue
		}

		message := Message{
			sender:  username,
			content: []byte(formatMsg),
			conn:    conn,
			msgDate: time.Now(),
		}

		// Store and broadcast the message
		if len(strings.Trim(msg, " ")) > 1 {
			s.msgStore = append(s.msgStore, message)
			s.msgChan <- message
		}
	}
}

func (s *Server) handleUserInput(conn net.Conn, msg string) []byte {
	switch {
	case strings.Contains(msg, "/name"):
		userName := strings.Fields(msg)[1]
		message := []byte(fmt.Sprintf("%s is now %s\n", s.clients[conn], userName))
		s.clients[conn] = userName
		s.clientInfomer(conn, message)
		return nil
	default:
		return []byte(fmt.Sprintf("%s\n", strings.TrimSpace(string(msg))))
	}
}

func (s *Server) addClient(conn net.Conn, userName string) {
	s.clients[conn] = userName[:len(userName)-1]
}

func (s *Server) broadcastMsg(conn net.Conn, msg []byte) {
	for client := range s.clients {
		if client != conn {
			timestamp := time.Now().Format("2006-01-02 15:04:05")
			message := fmt.Sprintf("[%v][%s]:%s\n", timestamp, s.clients[conn], msg)
			_, err := client.Write([]byte(message))
			if err != nil {
				fmt.Println("Error writing to connection:", err)
			}
		}
	}
}

func (s *Server) clientInfomer(conn net.Conn, msg []byte) {
	for client := range s.clients {
		if client != conn {
			message := fmt.Sprintf("\n%s\n", msg)
			_, err := client.Write([]byte(message))
			if err != nil {
				fmt.Println("Error writing to connection:", err)
			}
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
