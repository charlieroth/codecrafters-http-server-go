package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var directoryFlag *string

type Headers map[string]string

type Request struct {
	Method      string
	Path        string
	HttpVersion string
	Headers     Headers
	UserAgent   string
	Body        []byte
}

func ReadRequest(conn net.Conn) (Request, error) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return Request{}, err
	}

	line = strings.TrimSpace(line)
	line = strings.TrimSpace(line)
	parts := strings.Split(line, " ")
	if len(parts) < 3 {
		return Request{}, fmt.Errorf("invalid request line")
	}

	method, path, httpVersion := parts[0], parts[1], parts[2]
	headers := make(Headers)
	for {
		line, err = reader.ReadString('\n')
		if line == "\r\n" {
			break
		}
		parts = strings.Split(line, ":")
		headers[parts[0]] = strings.TrimSpace(parts[1])
	}

	body := []byte{}
	for reader.Buffered() > 0 {
		b, err := reader.ReadByte()
		if err != nil {
			return Request{}, fmt.Errorf("failed to read byte of body")
		}
		body = append(body, b)
	}

	req := Request{Method: method, Path: path, HttpVersion: httpVersion, Headers: headers, Body: body}
	return req, nil
}

func ReadEchoMessage(path string) string {
	_, msg, _ := strings.Cut(path, "/echo/")
	return strings.TrimSpace(msg)
}

func HandleFilePost(req Request, conn net.Conn, filename string) {
	filePath := filepath.Join(*directoryFlag, filename)
	file, err := os.Create(filePath)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 404 NotFound\r\n\r\n"))
		return
	}
	defer file.Close()

	file.Write(req.Body)
	conn.Write([]byte("HTTP/1.1 201 Created\r\n"))
	conn.Write([]byte("Content-Type: text/plain\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(req.Body))))
}

func HandleFileGet(conn net.Conn, filename string) {
	filePath := filepath.Join(*directoryFlag, filename)
	_, err := os.Stat(filePath)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 404 NotFound\r\n\r\n"))
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 404 NotFound\r\n\r\n"))
		return
	}

	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: application/octet-stream\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d", len(data))))
	conn.Write([]byte("\r\n\r\n"))
	conn.Write(data)
	conn.Write([]byte("\r\n"))
}

func HandleConnection(conn net.Conn) {
	defer conn.Close()

	req, err := ReadRequest(conn)
	if err != nil {
		fmt.Println("Failed to read request: ", err.Error())
		os.Exit(1)
	}

	if req.Path == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if req.Path == "/user-agent" {
		contentLength := fmt.Sprintf("Content-Length: %d", len(req.Headers["User-Agent"]))
		conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
		conn.Write([]byte("Content-Type: text/plain\r\n"))
		conn.Write([]byte(contentLength))
		conn.Write([]byte("\r\n\r\n"))
		conn.Write([]byte(req.Headers["User-Agent"]))
		conn.Write([]byte("\r\n"))
	} else if strings.HasPrefix(req.Path, "/echo") {
		message := ReadEchoMessage(req.Path)
		contentLength := fmt.Sprintf("Content-Length: %d", len(message))
		conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
		conn.Write([]byte("Content-Type: text/plain\r\n"))
		conn.Write([]byte(contentLength))
		conn.Write([]byte("\r\n\r\n"))
		conn.Write([]byte(message))
		conn.Write([]byte("\r\n"))
	} else if strings.HasPrefix(req.Path, "/files") {
		_, filename, _ := strings.Cut(req.Path, "/files/")
		filename = strings.TrimSpace(filename)
		if req.Method == "POST" {
			HandleFilePost(req, conn, filename)
		} else {
			HandleFileGet(conn, filename)
		}
	} else {
		conn.Write([]byte("HTTP/1.1 404 NotFound\r\n\r\n"))
	}
}

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	directoryFlag = flag.String("directory", ".", "directory of files to serve")
	flag.Parse() // parse os.Args[1:]
	fmt.Println("directory flag:", *directoryFlag)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}

		go HandleConnection(conn)
	}
}
