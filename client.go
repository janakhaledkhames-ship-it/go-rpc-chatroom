package main

import (
	"bufio"
	"fmt"
	"net/rpc"
	"os"
	"strings"
)

type Message struct {
    From string
    Text string
}

type Event struct {
    Type  string
    From  string
    Text  string
    Users []string
}

type Empty struct{}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your username: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	if name == "" {
		fmt.Println("Username cannot be empty.")
		return
	}

	conn, err := rpc.Dial("tcp", "localhost:1234")
	if err != nil {
		fmt.Println("Cannot connect to server:", err)
		return
	}
	defer conn.Close()

	var welcome string
	if err := conn.Call("Chat.Join", name, &welcome); err != nil {
		fmt.Println("Cannot join:", err)
		return
	}

	fmt.Println(welcome)
	fmt.Println("Commands: /users, /quit")
	fmt.Println("Type a message and press Enter.")

	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			var events []Event
			err := conn.Call("Chat.Poll", name, &events)
			if err != nil {
				return
			}

			for _, event := range events {
				switch event.Type {
				case "message":
					fmt.Printf("\n[%s]: %s\n", event.From, event.Text)
				case "join", "leave":
					fmt.Println("\n" + event.Text)
				}
			}
		}
	}()

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)

		if input == "/quit" {
			var reply string
			if err := conn.Call("Chat.Leave", name, &reply); err == nil {
				fmt.Println(reply)
			}
			break
		}

		if input == "/users" {
			var users []string
			if err := conn.Call("Chat.List", Empty{}, &users); err != nil {
				fmt.Println("Could not get user list:", err)
				continue
			}

			fmt.Println("Connected users:", strings.Join(users, ", "))
			continue
		}

		if input == "" {
			continue
		}

		var reply string
		err = conn.Call("Chat.Send", Message{
			From: name,
			Text: input,
		}, &reply)

		if err != nil {
			fmt.Println("Message failed:", err)
		}
	}

	fmt.Println("Client disconnected.")
}