package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", os.Args[1])
	if err != nil {
		fmt.Printf("Oops! %v\n", err)
		return
	}
	defer conn.Close()

	userInput := make(chan string)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			userInput <- input
		}
	}()

	go func() {
		reader := bufio.NewReader(conn)
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected:", err)
			return
		}
		fmt.Println(msg)
	}()
}
