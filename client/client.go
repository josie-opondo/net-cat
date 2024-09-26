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

	reader := bufio.NewReader(conn)

	// Read username prompt from server
	prompt, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from server:", err)
		return
	}
	fmt.Print(prompt)

	// Get username from the client (user input)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		username := scanner.Text()
		fmt.Fprintf(conn, username+"\n") // Send the username to the server
	}

	// Goroutine for receiving server messages
	go func() {
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Disconnected:", err)
				break
			}
			fmt.Print("👉 ", msg)
		}
	}()

	// Goroutine for sending user input
	for scanner.Scan() {
		input := scanner.Text()

		if input == "exit" {
			fmt.Fprintf(conn, "Goodbye!\n")
			conn.Close()
			os.Exit(0)
		}

		_, err := fmt.Fprintf(conn, input+"\n")
		if err != nil {
			fmt.Println("Error sending message:", err)
			return
		}
	}
}
