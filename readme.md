# go room

A simple chat room implemented in golang.

![](https://img.shields.io/badge/language-golang-brightgreen.svg?style=plastic)	![](https://img.shields.io/github/license/JameyWoo/goroom?logo=goroom)	



## Usage

### server

```
cd ./server
go run server.go 6666
```

Your service will listen on port 6666.

The default save location for file uploads is `./server/disk` 



## client

```
cd ./client
go run client.go 127.0.0.1 6666
```

The client will connect to the server with the IP address 127.0.0.1 and the port set to 6666.



## Protocol

### send messages

Users can send messages to the chat room, and everyone who logs in to the chat room can receive the message.

Messages other than legal command formats are sent as messages sent to chatroom



### commands

Users can execute commands as `%<command> args` to do something

such as:
```
%ls                         // View files in server
%exit                       // Exit client
%set-name username          // Set username
%download whyIsThat.pdf     // Download file
%upload test.go             // Upload file
```

Because TCP transmits a stream of bytes, before sending each message, use a 4-byte data to declare the length of the message to be sent to avoid sticky packets.



