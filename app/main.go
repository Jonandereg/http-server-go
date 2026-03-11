package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type Request struct {
	Method string
	URL    string
	Proto  string
}

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	fmt.Println("Listening on 0.0.0.0:4221")
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		req := make([]byte, 1024)
		if _, err := conn.Read(req); err != nil {
			fmt.Println("Error reading request: ", err.Error())
		}
		fmt.Println("Request received: ", string(req))

		parsedReq := parseHTTPRequest(string(req))
		if parsedReq.URL == "/" {
			if _, err := conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
				fmt.Println("error writing to connection", err.Error())
			}

		} else {
			if _, err := conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n")); err != nil {
				fmt.Println("error writing to connection", err.Error())
			}
		}

		conn.Close()
	}

}

func parseHTTPRequest(s string) Request {
	crlf := "\r\n"
	splitReq := strings.Split(s, crlf)
	requestLineArr := strings.Split(splitReq[0], " ")
	return Request{
		Method: requestLineArr[0],
		URL:    requestLineArr[1],
		Proto:  requestLineArr[2],
	}
}
