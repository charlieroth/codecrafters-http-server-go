package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	rawBody := make([]byte, 2048)
	_, err = conn.Read(rawBody)
	if err != nil {
		fmt.Println("Failed to read bytes from connection: ", err.Error())
		os.Exit(1)
	}

	body := string(rawBody[:])
	bodyLines := strings.Split(body, "\r\n")
	startLine := strings.SplitN(bodyLines[0], " ", 3)

	path := startLine[1]
	var response []byte
	if path == "/" {
		response = []byte("HTTP/1.1 200 OK\r\n\r\n")
	} else if path != "/" {
		pathParts := strings.SplitN(path, "/", 3)
		command := pathParts[1]
		message := pathParts[2]
		if command == "echo" {
			contentLength := len(message)
			response = []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s\r\n\r\n", contentLength, message))
		} else {
			response = []byte("HTTP/1.1 404 NotFound\r\n\r\n")
		}
	} else {
		response = []byte("HTTP/1.1 404 NotFound\r\n\r\n")
	}

	_, err = conn.Write(response)
	if err != nil {
		fmt.Println("Failed to write bytes to connection: ", err.Error())
		os.Exit(1)
	}
}
