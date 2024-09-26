package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:4080") // Ensure the correct port
	if err != nil {
		fmt.Printf("Oops! Couldn't connect to the server: %v\n", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Wait for the server's prompt asking for the name
	prompt, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from server:", err)
		return
	}
	fmt.Print(prompt)

	// Get user's name and send it to the server
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		name := scanner.Text()
		fmt.Fprintf(conn, name+"\n")
	}

	// Channel to handle user input
	userInput := make(chan string)

	// Goroutine to handle user input
	go func() {
		for scanner.Scan() {
			input := scanner.Text()
			if input == "exit" {
				fmt.Println("Exiting the chat. Goodbye!")
				close(userInput)
				return
			}
			userInput <- input
		}
	}()

	// Goroutine to listen for messages from the server
	go func() {
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Disconnected from server.")
				return
			}
			fmt.Print(msg) // Print messages from the server
		}
	}()

	// Main loop to handle sending user input to the server
	for {
		select {
		case input, ok := <-userInput:
			if !ok {
				return // User closed the input, exit the loop
			}
			_, err := fmt.Fprintf(conn, input+"\n")
			if err != nil {
				fmt.Println("Failed to send message. Disconnecting...")
				return
			}
		}
	}
}
