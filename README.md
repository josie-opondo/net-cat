# TCP Chat Application (NetCat Recreation)

This project is a recreation of the NetCat utility, designed to implement a Server-Client architecture for group chat functionality. It features a TCP-based communication system that enables multiple clients to connect to a server, exchange messages, and participate in real-time conversations.

## Project Overview

NetCat (`nc`) is a command-line tool that facilitates reading and writing data across network connections using TCP or UDP protocols. This project emulates its core functionalities and extends them into a multi-client chatroom application. The application includes features such as connection handling, user identification, and message broadcasting.

## Features

1. **Server-Client Architecture**:  
   - The server can handle multiple client connections via TCP (1-to-many relationship).
   - Maximum of 10 simultaneous connections.

2. **Client Identification**:  
   - Clients must provide a name to join the chat.
   - All messages include a timestamp and the sender’s name:  
     `[YYYY-MM-DD HH:MM:SS][client.name]:[client.message]`.

3. **Message Handling**:  
   - Messages are broadcasted to all connected clients.
   - Empty messages are ignored.
   - New clients receive the chat history upon joining.

4. **Notifications**:  
   - Clients are informed when a new client joins or exits the chatroom.

5. **Resilience**:  
   - If a client disconnects, the chatroom remains functional for others.

6. **Default and Custom Port**:  
   - Default port: `8989`.  
   - If a custom port is not specified, the program provides a usage message:  
     `[USAGE]: ./TCPChat $port`.

7. **Welcome Message**:  
   - Includes a Linux logo and prompts clients to enter their name.

## Instructions

### Prerequisites
- The project is written in Go.

### Server Setup
1. Run the server on the default port:  
   ```bash
   $ go run .
   Listening on the port :8989
   ```
2. Run the server on a custom port:  
   ```bash
   $ go run . 2525
   Listening on the port :2525
   ```

### Client Connection
1. Connect to the server via `nc`:
   ```bash
   $ nc $IP $port
   ```

2. Upon connection, the client will receive:
   ```plaintext
   Welcome to TCP-Chat!
            _nnnn_
           dGGGGMMb
          @p~qp~~qMb
          M|@||@) M|
          @,----.JM|
         JS^\__/  qKL
        dZP        qKRb
       dZP          qKKb
      fZP            SMMb
      HZM            MMMM
      FqM            MMMM
    __| ".        |\dS"qML
    |    `.       | `' \Zq
   _)      \.___.,|     .'
   \____   )MMMMMP|   .'
        `-'       `--'
   [ENTER YOUR NAME]:
   ```

### Example Interaction
#### Client 1 (Yenlik):
```plaintext
[ENTER YOUR NAME]: Yenlik
[2020-01-20 16:03:43][Yenlik]:hello
[2020-01-20 16:04:10][Yenlik]:
Lee has joined our chat...
[2020-01-20 16:04:32][Lee]:Hi everyone!
[2020-01-20 16:04:50][Lee]:alright, see ya!
[2020-01-20 16:04:57][Yenlik]:bye-bye!
Lee has left our chat...
```

#### Client 2 (Lee):
```plaintext
[ENTER YOUR NAME]: Lee
[2020-01-20 16:03:43][Yenlik]:hello
[2020-01-20 16:04:32][Lee]:Hi everyone!
[2020-01-20 16:04:35][Lee]:How are you?
[2020-01-20 16:04:41][Yenlik]:great, and you?
[2020-01-20 16:04:50][Lee]:alright, see ya!
```

### Error Handling
- Names must be non-empty to join the chat.
- Empty messages will not be transmitted

## Good Practices
- Go routines are used for concurrency.
- Implementation of channels for data synchronization.

## Usage Examples
```bash
$ go run .            # Start server on default port
$ go run . 2525       # Start server on port 2525
$ nc localhost 2525   # Connect client to server
```

---

Authors

[Josephine Opondo](https://github.com/josie-opondo)

[Raymond Muiruri](https://github.com/rayjonesjay)

[Andrew Osindo](https://github.com/andyosyndoh)