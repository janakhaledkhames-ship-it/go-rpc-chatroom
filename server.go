package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"
)

const serverAddress = "localhost:1234"

type Message struct {
	From string
	Text string
}

type Event struct {
	Type    string
	From    string
	Text    string
	Users   []string
}

type Empty struct{}

type ChatServer struct {
	mu      sync.Mutex
	users   map[string]*Client
	history []Event
}

type Client struct {
	Name    string
	Events  []Event
	Notify  chan struct{}
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		users: make(map[string]*Client),
	}
}

func (s *ChatServer) userListLocked() []string {
	names := make([]string, 0, len(s.users))
	for name := range s.users {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *ChatServer) broadcastLocked(event Event, exclude string) {
	event.Users = s.userListLocked()

	for name, client := range s.users {
		if name == exclude {
			continue
		}
		client.Events = append(client.Events, event)
		close(client.Notify)
		client.Notify = make(chan struct{})
	}
}

func (s *ChatServer) Join(name string, reply *string) error {
	if name == "" {
		return errors.New("username cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[name]; exists {
		return errors.New("username already exists")
	}

	client := &Client{
		Name:   name,
		Notify: make(chan struct{}),
	}
	s.users[name] = client

	s.broadcastLocked(Event{
		Type: "join",
		From: name,
		Text: name + " joined the chat",
	}, name)

	*reply = "Welcome, " + name
	log.Printf("%s joined", name)
	return nil
}

func (s *ChatServer) Send(message Message, reply *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[message.From]; !exists {
		return errors.New("user is not connected")
	}
	if message.Text == "" {
		return errors.New("message cannot be empty")
	}

	s.broadcastLocked(Event{
		Type: "message",
		From: message.From,
		Text: message.Text,
	}, message.From)

	*reply = "Message delivered"
	log.Printf("Message from %s", message.From)
	return nil
}

func (s *ChatServer) List(_ Empty, reply *[]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	*reply = s.userListLocked()
	return nil
}

func (s *ChatServer) Poll(name string, reply *[]Event) error {
	s.mu.Lock()

	client, exists := s.users[name]
	if !exists {
		s.mu.Unlock()
		return errors.New("user is not connected")
	}

	events := append([]Event(nil), client.Events...)
	client.Events = nil
	notify := client.Notify
	s.mu.Unlock()

	// Wait for new events or check periodically for shutdown.
	select {
	case <-notify:
	case <-time.After(20 * time.Second):
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	client, exists = s.users[name]
	if !exists {
		return errors.New("user disconnected")
	}

	*reply = append(events, client.Events...)
	client.Events = nil
	return nil
}

func (s *ChatServer) Leave(name string, reply *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[name]; !exists {
		*reply = "Already disconnected"
		return nil
	}

	delete(s.users, name)
	s.broadcastLocked(Event{
		Type: "leave",
		From: name,
		Text: name + " left the chat",
	}, name)

	*reply = "Goodbye, " + name
	log.Printf("%s left", name)
	return nil
}

func main() {
	server := NewChatServer()

	if err := rpc.RegisterName("Chat", server); err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", serverAddress)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Chat server listening on", serverAddress)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		log.Println("Shutting down server...")
		listener.Close()
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		go rpc.ServeConn(conn)
	}

	fmt.Println("Server stopped")
}