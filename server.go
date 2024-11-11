package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Server struct {
	listenAddr  string
	ln          net.Listener
	msgChan     chan Message
	clients     map[net.Conn]string
	sem         chan struct{}
	msgStore    []Message
	shutdown    chan struct{}       // Shutdown channel
	rooms       map[string][]Client // Map to store clients in rooms
	clientRooms map[net.Conn]string // track current room of each client
}

type Client struct {
	conn     net.Conn
	userName string
	room     string
}

type Message struct {
	sender  string
	content []byte
	conn    net.Conn
	msgDate time.Time
}

func NewServer(port string) (*Server, error) {
	return &Server{
		listenAddr:  port,
		msgChan:     make(chan Message, 10),
		clients:     make(map[net.Conn]string),
		sem:         make(chan struct{}, 10),
		msgStore:    make([]Message, 0),
		shutdown:    make(chan struct{}),       // Initialize the shutdown channel
		rooms:       make(map[string][]Client), // intialize the rooms map
		clientRooms: make(map[net.Conn]string),
	}, nil
}

func (s *Server) Logo() (string, error) {
	logo := "\033[34m" + // Start blue background
		"          _nnnn_\n" +
		"         \033[32mdGGGGMMb\033[34m\n" + // Green
		"        \033[32m@p~qp~~qMb\033[34m\n" + // Green
		"        \033[32mM|\033[33m@\033[32m||\033[33m@) M|\033[34m\n" + // Green with yellow for '@'
		"        \033[32m@,----.JM|\033[34m\n" + // Green
		"       \033[32mJS^\\__/  qKL\033[34m\n" + // Green
		"      \033[32mdZP        qKRb\033[34m\n" + // Green
		"     \033[32mdZP          qKKb\033[34m\n" + // Green
		"    \033[32mfZP            SMMb\033[34m\n" + // Green
		"    \033[32mHZM            MMMM\033[34m\n" + // Green
		"    \033[32mFqM            MMMM\033[34m\n" + // Green
		" \033[34m__\033[32m | \".        |\\dS\"qML\033[34m\n" + // Green with blue
		" \033[34m|    `.        | `' \\Zq\033[34m\n" +
		" \033[34m_)      \\.___.,|     .'\033[34m\n" +
		" \033[34m\\____   )MMMMMP|   .'\033[34m\n" +
		"      `-'       `--'\033[0m" // Reset colors
	return string(logo), nil
}

func (s *Server) Start(ctx context.Context) error {
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

	// Listen for shutdown signal from the context
	<-ctx.Done()

	// Perform shutdown actions
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
				return
			default:
				fmt.Printf("Error accepting connection: %v\n", err)
				continue
			}
		}

		// Semaphore to limit connections
		select {
		case s.sem <- struct{}{}:
			go s.handleClient(conn)
		default:
			conn.Write([]byte("Chatroom is at max capacity. Try later...\n"))
			conn.Close()
		}
	}
}

var UserNames = make(map[string]bool)

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		s.removeClient(conn)
		conn.Close()
		<-s.sem
	}()

	logo, _ := s.Logo()
	welcomeMessage := fmt.Sprintf("Welcome to TCP-Chat!\n%s\n[ENTER YOUR NAME]: ", logo)
	conn.Write([]byte(welcomeMessage))

	userName, _ := bufio.NewReader(conn).ReadString('\n')
	if len(strings.TrimSpace(userName)) < 3 {
		conn.Write([]byte("Enter a valid name. Disconnecting...\n"))
		return
	}

	userName = strings.TrimSpace(userName)
	// check if user name exist, purpose is to ensure each user has a unique username

	ok := UserNames[userName]
	if !ok {
		UserNames[userName] = true
	} else {
		// fmt.Println("username already exists try another name")
		// msg := "%s already existed so you were assigned a new nickname [%s]"
		randInt := rand.Intn(10)
		userName = fmt.Sprintf("%s%d", userName, randInt)
	}

	client := Client{
		conn:     conn,
		userName: userName,
		room:     "room1",
	}

	s.addClient(conn, client)

	// s.addClient(conn, userName)
	// create a new room for the client
	roomName := fmt.Sprintf("room1_%s", s.listenAddr)

	s.joinRoom(client, roomName)

	conn.Write([]byte(fmt.Sprintf("Welcome, %s!\nUse /help for more options.\n", userName)))
	s.clientInfomer(conn, []byte(fmt.Sprintf("%s has joined the chat!\n", userName)), true)

	for _, msg := range s.msgStore {
		timestamp := msg.msgDate.Format("2006-01-02 15:04:05")
		message := fmt.Sprintf("[%v][%s]:%s", timestamp, msg.sender, string(msg.content))
		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Error writing to connection:", err)
		}
	}

	s.readConn(client)
}

func (s *Server) readConn(client Client) {
	reader := bufio.NewReader(client.conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			s.clientInfomer(client.conn, []byte(fmt.Sprintf("\n%s has left the chat\n", client.userName)), true)
			break
		}

		formatMsg := s.handleUserInput(client, msg)
		if formatMsg == nil {
			continue
		}

		message := Message{
			sender:  client.userName,
			content: []byte(formatMsg),
			conn:    client.conn,
			msgDate: time.Now(),
		}

		// Store and broadcast the message
		if len(strings.Trim(msg, " ")) > 1 {
			s.msgStore = append(s.msgStore, message)
			s.msgChan <- message
		}
	}
}

func (s *Server) handleUserInput(client Client, msg string) []byte {
	switch {
	case strings.Contains(msg, "/name"):
		if len(strings.Fields(msg)) < 2 {
			message := []byte("Enter new name after /name\n")
			s.clientInfomer(client.conn, message, false)
			return nil
		}
		newUserName := strings.Fields(msg)[1]
		oldUserName := client.userName
		client.userName = newUserName
		s.clients[client.conn] = client.userName
		message := []byte(fmt.Sprintf("%s is now %s\n", oldUserName, newUserName))
		s.broadcastToRoom(client.conn, message, client.conn)

		// Confirm the name change to the client who requested it
		confirmation := fmt.Sprintf("\nSuccess! You are now %s\n\n", newUserName)
		s.clientInfomer(client.conn, []byte(confirmation), false)
		return nil

	case strings.Contains(msg, "/users"):
		message := "\nBuddies currently in the chat:\n"
		for user := range s.clients {
			message += fmt.Sprintf("%s\n", s.clients[user])
		}
		s.clientInfomer(client.conn, []byte(message), false)
		return nil

	case strings.Contains(msg, "/help"):
		message := "\nAvailable commands:\n/name [new-name]: Change your name\n/users: See who's in the chat\n/help: Display this log of available commands\n/quit: Leave the chat\n/join [room-name]: Join a specific room\n/leave: Leave your current room\n/rooms: List all available rooms\n/rooms [room-name]: List members in a specific room\n\n"
		s.clientInfomer(client.conn, []byte(message), false)
		return nil

	case strings.Contains(msg, "/quit"):
		message := "\nExiting the chat..."
		s.clientInfomer(client.conn, []byte(message), false)
		s.leaveRoom(client.conn)
		client.conn.Close()
		return nil

	case strings.HasPrefix(msg, "/join"):
		msgs := strings.Fields(msg)
		if len(msgs) > 1 {
			roomName := strings.TrimSpace(msgs[1])
			if roomName == "" {
				s.clientInfomer(client.conn, []byte("Usage: /join [room-name]\n"), false)
				return nil
			}
			s.joinRoom(client, roomName)
			return nil
		}

	case strings.Contains(msg, "/leave"):
		s.leaveRoom(client.conn)
		return nil

	case strings.Contains(msg, "/rooms"):
		args := strings.Fields(msg)
		if len(args) == 1 {
			s.listRooms(client.conn)
		} else {
			room := strings.TrimSpace(args[1])
			s.listRoomMembers(client.conn, room)
		}

	default:
		return []byte(fmt.Sprintf("%s\n", strings.TrimSpace(string(msg))))
	}

	return nil
}

func (s *Server) leaveRoom(conn net.Conn) {
	currentRoom := s.clientRooms[conn]

	// get list of clients in the current room
	clients, roomExists := s.rooms[currentRoom]
	if !roomExists {
		s.clientInfomer(conn, []byte("Room does not exist.\n"), false)
		return
	}

	// find and remove the client from the room's client slice
	for i, client := range clients {
		if client.conn == conn {
			// remove the client from the room slice
			s.rooms[currentRoom] = append(clients[:i], clients[i+1:]...)

			// remove the client from clientRooms mapping
			delete(s.clientRooms, conn)

			// notify the client that they have left the room
			s.clientInfomer(conn, []byte(fmt.Sprintf("You have left the room: %s\n", currentRoom)), false)

			// notify others
			s.broadcastToRoom(conn, []byte(fmt.Sprintf("%s has left the room!", s.clients[conn])), conn)

			break
		}
	}

	// if the room is now empty, delete it
	if len(s.rooms[currentRoom]) == 0 {
		delete(s.rooms, currentRoom)
	}
}

func (s *Server) broadcastToRoom(sender net.Conn, msg []byte, exclude net.Conn) {
	currentRoom, ok := s.clientRooms[sender]
	if !ok {
		return
	}

	for _, client := range s.rooms[currentRoom] {

		if client.conn == sender || client.conn == exclude {
			continue
		}

		timestamp := TimeFormat()
		clearscreen := "\033[F\033[K"
		// message := fmt.Sprintf("[%v][%s]:%s", timestamp, client.userName, msg)

		message := fmt.Sprintf("%v[%v][%s]:%s", clearscreen, timestamp, client.userName, msg)
		s.Logs(message)
		_, err := client.conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Error writing to connection:", err)
		}
	}
}

func (s *Server) joinRoom(client Client, roomName string) {
	// leave the current room if the client is in one
	s.leaveRoom(client.conn)

	// add the client to the new room
	s.rooms[roomName] = append(s.rooms[roomName], client)
	s.clientRooms[client.conn] = roomName
	s.clientInfomer(client.conn, []byte(fmt.Sprintf("You have joined room: %s\n", roomName)), false)

	// notify the other clients in the room
	s.broadcastToRoom(client.conn, []byte(fmt.Sprintf("%s has joined the room!\n", client.userName)), nil)
}

// for logging errors to a file, need to see whats happening when program is running
func Logger(functionName string, lineNumber int, data interface{}) {
	file := "logger.log"
	fd, _ := os.OpenFile(file, os.O_APPEND|os.O_CREATE, 0o744)
	defer fd.Close()
	fd.WriteString(fmt.Sprintf("function-> %v\nline Number %d\ndata %v\n", functionName, lineNumber, data))
}

func (s *Server) addClient(conn net.Conn, client Client) {
	s.clients[conn] = client.userName
}

func (s *Server) broadcastMsg(conn net.Conn, msg []byte) {
	for client := range s.clients {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		clearscreen := "\033[F\033[K"
		message := fmt.Sprintf("[%v][%s]:%s", timestamp, s.clients[conn], msg)
		s.Logs(message)
		if client == conn {
			message = fmt.Sprintf("%v[%v][%s]:%s", clearscreen, timestamp, s.clients[conn], msg)
		}
		_, err := client.Write([]byte(message))
		if err != nil {
			fmt.Println("Error writing to connection:", err)
		}
	}
}

func (s *Server) clientInfomer(conn net.Conn, msg []byte, broadcast bool) {
	if broadcast {
		for client := range s.clients {
			if client != conn {
				message := fmt.Sprintf("\r%s\n", msg)
				s.Logs(message)
				_, err := client.Write([]byte(message))
				if err != nil {
					fmt.Println("Error writing to connection:", err)
				}
			}
		}
	} else {
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Println("Error writing to connection:", err)
		}
	}
}

func TimeFormat() string {
	return time.Now().Format("2006-01-02 15:04:05")
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

func (s *Server) listRooms(conn net.Conn) {
	var rooms []string
	for room := range s.rooms {
		rooms = append(rooms, room)
	}
	conn.Write([]byte(fmt.Sprintf("available rooms: %s\n", strings.Join(rooms, ", "))))
}

func (s *Server) listRoomMembers(conn net.Conn, room string) {
	clients, exists := s.rooms[room]

	if !exists {
		conn.Write([]byte(fmt.Sprintf("Room %s does not exist.\n", room)))
		return
	}

	var members []string
	for _, client := range clients {
		members = append(members, client.userName)
	}
	conn.Write([]byte(fmt.Sprintf("Members in %s: %s\n", room, strings.Join(members, ", "))))
}

func Check(arg string) bool {
	for _, char := range arg {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

var mu sync.Mutex

func (s *Server) Logs(msg string) {
	mu.Lock()
	defer mu.Unlock()
	filename := "history.log"
	fileDescriptor, err:= os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	defer fileDescriptor.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	fileDescriptor.WriteString(msg)
}
func main() {
	var port string
	args := os.Args
	if len(args) == 1 {
		// default port
		port = ":8989"
	} else if len(args) == 2 && Check(args[1]) {
		port = ":" + args[1]
	} else {
		fmt.Println("[USAGE]: ./TCPChat $port")
		return
	}

	server, err := NewServer(port)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Server running on port: ", port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			if input == "exit" {
				cancel()
				fmt.Println("\nServer shutting down...")
				break
			}
		}
	}()

	server.Start(ctx)
}
