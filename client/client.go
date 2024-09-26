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

	userInput := make(chan string)

	// Goroutine for reading user input
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			if input == "exit" {
				close(userInput)
				return
			}
			userInput <- input
		}
	}()

	// Goroutine for reading server messages
	go func() {
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Disconnected from server.")
				return
			}
			fmt.Println("👉 ", msg)
		}
	}()

	// Main loop for sending user input to the server
	for input := range userInput {
		fmt.Fprintf(conn, input+"\n")
	}

	fmt.Println("Goodbye buddies!")
}
