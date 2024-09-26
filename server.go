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
	clients    map[net.Conn]struct{}
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
			go func() {
				defer func() {
					conn.Close()
					s.removeClient(conn)
					<-s.sem // Release token
				}()

				conn.Write([]byte("Hey buddy, what's your name? "))
				userName, _ := bufio.NewReader(conn).ReadString('\n')

				s.addClient(conn)

				s.broadcastMsg(conn, []byte(fmt.Sprintf("%s has joined the chat!", userName)))

				for _, mess := range s.msgStore {
					conn.Write([]byte(mess.sender))
					conn.Write([]byte(mess.content))
				}

				s.readConn(conn, strings.TrimSpace(userName))
				log.Printf("received connection: %s", conn.RemoteAddr())
			}()
		default:
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
			s.broadcastMsg(conn, []byte(fmt.Sprintf("\nOops! %s disconnected", username)))
			conn.Close()
			return
		}
		msg := buff[:n]
		formatMsg := []byte(fmt.Sprintf("\n%s: %s\n", username, strings.TrimSpace(string(msg))))

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

func (s *Server) addClient(conn net.Conn) {
	s.clients[conn] = struct{}{}
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
	fmt.Println("Server running on port: ", port)
	go func() {
		for msg := range server.msgChan {
			server.broadcastMsg(msg.conn, msg.content)
		}
	}()
	server.Start()
}
