package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

const reconnectAttempts = 3

// Client represents a chat client
type Client struct {
	conn     net.Conn
	username string
	input    chan string
}

// NewClient initializes a new Client and handles connection
func NewClient(serverAddr string) (*Client, error) {
	conn, err := connectToServer(serverAddr)
	if err != nil {
		return nil, err
	}

	client := &Client{
		conn:  conn,
		input: make(chan string),
	}

	return client, nil
}

// Start begins the client interaction
func (c *Client) Start() {
	defer c.conn.Close()

	reader := bufio.NewReader(c.conn)
	if err := c.readServerPrompt(reader); err != nil {
		return
	}

	// Start user input and server listening in separate goroutines
	go c.handleUserInput()
	go c.listenForServerMessages(reader)

	c.mainLoop()
}

// readServerPrompt handles the initial prompt from the server asking for the name
func (c *Client) readServerPrompt(reader *bufio.Reader) error {
	prompt, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from server:", err)
		return err
	}
	fmt.Print(prompt)

	// Get user's name and send it to the server
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		c.username = scanner.Text()
		_, err = fmt.Fprintf(c.conn, c.username+"\n")
		if err != nil {
			fmt.Println("Error sending username:", err)
			return err
		}
	}
	return nil
}

// handleUserInput handles reading user input and sends it to the input channel
func (c *Client) handleUserInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		if input == "exit" {
			fmt.Println("Exiting the chat. Goodbye!")
			close(c.input)
			return
		}
		c.input <- input
	}
}

// listenForServerMessages listens for incoming messages from the server and displays them
func (c *Client) listenForServerMessages(reader *bufio.Reader) {
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected from server.")
			close(c.input)
			return
		}
		fmt.Print(msg)
	}
}

// mainLoop listens to user inputs and handles sending messages to the server
func (c *Client) mainLoop() {
	for input := range c.input {
		if input == "/users" {
			c.sendMessage("/users\n")
		} else if input == "/help" {
			displayHelp()
		} else {
			c.sendMessage(input + "\n")
		}
	}
}

// sendMessage sends a message to the server
func (c *Client) sendMessage(message string) {
	_, err := fmt.Fprint(c.conn, message)
	if err != nil {
		fmt.Println("Failed to send message. Disconnecting...")
		close(c.input)
	}
}

// connectToServer handles connection to the server with retry logic
func connectToServer(serverAddr string) (net.Conn, error) {
	var conn net.Conn
	var err error
	for i := 1; i <= reconnectAttempts; i++ {
		conn, err = net.Dial("tcp", serverAddr)
		if err == nil {
			return conn, nil
		}
		fmt.Printf("Attempt %d: Unable to connect to server. Retrying...\n", i)
		time.Sleep(2 * time.Second) // Wait before retrying
	}
	return nil, err
}

// displayHelp prints the available commands
func displayHelp() {
	fmt.Println("Available commands:")
	fmt.Println("/users  - List online users")
	fmt.Println("/help   - Show this help message")
	fmt.Println("/exit   - Exit the chat")
}

func main() {
	serverAddr := "localhost:4080"

	// Initialize client
	client, err := NewClient(serverAddr)
	if err != nil {
		fmt.Printf("Couldn't connect to server after %d attempts: %v\n", reconnectAttempts, err)
		return
	}

	client.Start()
}
