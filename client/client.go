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
	prompt, err := reader.ReadString(':')
	if err != nil {
		fmt.Println("Error reading from server: ", err)
		return err
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

func (c *Client) handleUserInput() {
    scanner := bufio.NewScanner(os.Stdin)

    for scanner.Scan() {
        input := scanner.Text()

        // Handle special commands
        if input == "/users" {
			c.resetCursor(input)
            c.sendMessage("/users\n")
        } else if input == "/help" {
			c.resetCursor(input)
            c.displayHelp()
        } else if input == "/exit" {
            fmt.Println("Exiting the chat. Goodbye...")
            close(c.input) // close the input channel only on exit
            return
        } else {
			c.resetCursor(input)
            // Send the input to the input channel
            c.input <- input
        }
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading user input:", err)
        close(c.input) // close the channel in case of error
    }
}

func (c *Client) resetCursor(msg string) {
    // Move cursor up one line and clear the line
    fmt.Print("\033[F\033[K")
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("[%v][%s]:%s\n", timestamp, c.userName, msg)
	fmt.Print(message)
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


func (c *Client) displayHelp() {
	fmt.Println("Available commands:")
	fmt.Println("/users - List online users")
	fmt.Println("/help - Show help")
	fmt.Println("/exit - Exit the chat")
}

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
