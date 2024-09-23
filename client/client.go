package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <address>")
		return
	}

	conn, err := net.Dial("tcp", os.Args[1])
	if err != nil {
		fmt.Printf("Oops! %v\n", err)
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
		reader := bufio.NewReader(conn)
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Disconnected:", err)
				break
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
