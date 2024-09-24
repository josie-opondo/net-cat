package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Printf("Oops! %v\n", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Wait for the server's prompt for the name
	prompt, _ := reader.ReadString('\n')
	fmt.Print(prompt)

	// Ask for the user's name and send it to the server
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		name := scanner.Text()
		fmt.Fprintf(conn, name+"\n")
	}

	// Goroutine to listen for messages from the server
	go func() {
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Disconnected from server:", err)
				break
			}
			fmt.Print(msg)
		}
	}()

	// Main loop for reading and sending user messages
	for scanner.Scan() {
		input := scanner.Text()
		if input == "exit" {
			break
		}
		fmt.Fprintf(conn, input+"\n")
	}

	fmt.Println("Goodbye!")
}
