package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const reconnectAttempts = 3

type Client struct {
	conn     net.Conn
	userName string
	input    chan string
}

// NewClient creates a new client instance and connects it to the server
func NewClient(serverAdr string) (*Client, error) {
	conn, err := connectToServer(serverAdr)
	if err != nil {
		return nil, err
	}

	client := &Client{
		conn:  conn,
		input: make(chan string),
	}

	return client, nil
}

// connectToServer tries to establish a connection to the server with retry logic
func connectToServer(serverAdr string) (net.Conn, error) {
	var conn net.Conn
	var err error

	for i := 1; i <= reconnectAttempts; i++ {
		conn, err = net.Dial("tcp", serverAdr)
		if err == nil {
			return conn, nil
		}
		fmt.Printf("Connection failed. Attempting: %d\n", i)
		time.Sleep(2 * time.Second)
	}
	return nil, err
}

// Start manages the main client logic and handles communication
func (c *Client) Start() {
	defer c.conn.Close()

	reader := bufio.NewReader(c.conn)
	if err := c.readServerPrompt(reader); err != nil {
		return
	}

	go c.handleUserInput()
	go c.listenForServerMessages(reader)

	c.mainLoop()
}

// readServerPrompt reads the initial server prompt (e.g., username prompt) and sends the username
func (c *Client) readServerPrompt(reader *bufio.Reader) error {
	prompt, err := reader.ReadString(':')
	if err != nil {
		fmt.Println("Oops, the chatroom is at max capacity. Try again later... ")
		os.Exit(0)
	}
	fmt.Print(prompt + " ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		c.userName = scanner.Text()
		_, err = fmt.Fprintf(c.conn, c.userName+"\n")
		if err != nil {
			fmt.Println("Error sending username: ", err)
			return err
		}
	}

	return nil
}

// handleUserInput captures user input from the console and sends it to the input channel
func (c *Client) handleUserInput() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		input := scanner.Text()
		if input != strings.Trim("\n", " ") {
			c.input <- input
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading user input:", err)
		close(c.input) // close the channel in case of error
	}
}

// listenForServerMessages continuously listens for and prints messages from the server
func (c *Client) listenForServerMessages(reader *bufio.Reader) {
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconected from the server. Exiting...")
			os.Exit(0)
		}

		fmt.Print(msg)
	}
}

// sendMessage sends a user message to the server
func (c *Client) sendMessage(msg string) {
	writer := bufio.NewWriter(c.conn)

	// Ensure each message ends with a newline (or another delimiter expected by the server)
	_, err := writer.WriteString(msg + "\n")
	if err != nil {
		fmt.Println("Failed to send message:", err)
		close(c.input)
		return
	}

	// Flush the writer to ensure the message is sent
	err = writer.Flush()
	if err != nil {
		fmt.Println("Failed to flush message:", err)
		close(c.input)
	}
}

// mainLoop continuously processes user input and sends messages to the server
func (c *Client) mainLoop() {
	for input := range c.input {
		c.sendMessage(input)
	}
	fmt.Println("Input channel closed. Exiting...")
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run . <hostIp:port>")
		return
	}

	serverAdr := os.Args[1]
	client, err := NewClient(serverAdr)
	if err != nil {
		fmt.Printf("Could not connect to Server after attempting %d\n %v", reconnectAttempts, err)
		return
	}

	client.Start()
}
