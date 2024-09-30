package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

const reconnectAttempts = 3

type Client struct {
	conn     net.Conn
	userName string
	input    chan string
}

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

func (c *Client) readServerPrompt(reader *bufio.Reader) error {
	prompt, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from server: ", err)
		return err
	}
	fmt.Print(prompt)

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

func (c *Client) handleUserInput() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		input := scanner.Text()

		if input == "/exit" {
			fmt.Println("Exiting the chat. Goodbye...")
			close(c.input)
			return
		}
		c.input <- input
	}
}

func (c *Client) listenForServerMessages(reader *bufio.Reader) {
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconected from the server")
			close(c.input)
			return
		}

		fmt.Print(msg)
	}
}

func (c *Client) sendMessage(msg string) {
	_, err := fmt.Fprint(c.conn, msg)
	if err != nil {
		fmt.Println("Failed to send message.")
		close(c.input)
	}
}

func (c *Client) displayHelp() {
	fmt.Println("Available commands:")
	fmt.Println("/users - List online users")
	fmt.Println("/help - Show help")
	fmt.Println("/exit - Exit the chat")
}

func (c *Client) mainLoop() {
	for input := range c.input {
		if input == "/users" {
			c.sendMessage("/users\n")
		} else if input == "/help" {
			c.displayHelp()
		} else {
			c.sendMessage(input)
		}
	}
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
