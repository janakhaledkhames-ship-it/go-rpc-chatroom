# Go RPC Chat Room

A simple multi-client chat room built with Go using net/rpc.

## Features
- Multiple clients can connect to one server.
- Users can send messages to other connected users.
- Displays connected users.
- Notifies users when someone joins or leaves.
- Rejects duplicate usernames.
- Uses goroutines and mutexes for concurrent access.

## Requirements
- Go installed on your computer.

## How to Run

### 1. Start the Server
Open a terminal in the project folder and run:

```bash
go run server.go
```

### 2. Start a Client
Open another terminal and run:

```bash
go run client.go
```

Enter a username when prompted.

Run the client command in additional terminals to connect more users.

## Client Commands
- `/users` - Display connected users.
- `/quit` - Leave the chat.

## Technologies
- Go
- net/rpc
- TCP
- Goroutines
- Mutexes