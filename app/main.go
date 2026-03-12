package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Request struct {
	Method  string
	URL     string
	Proto   string
	headers map[string]string
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
		go handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	req := make([]byte, 1024)
	if _, err := conn.Read(req); err != nil {
		fmt.Println("Error reading request: ", err.Error())
	}
	fmt.Println("Request received: ", string(req))

	parsedReq := parseHTTPRequest(string(req))

	router(parsedReq, conn)
}

func parseHTTPRequest(s string) Request {
	crlf := "\r\n"
	splitReq := strings.Split(s, crlf)
	requestLineArr := strings.Split(splitReq[0], " ")
	headersArr := splitReq[1:]
	headers := make(map[string]string)
	for _, header := range headersArr {
		if header == "" {
			continue
		}
		headerArr := strings.Split(header, ":")
		if len(headerArr) == 2 {
			headers[strings.ToLower(headerArr[0])] = strings.TrimSpace(headerArr[1])
		}
	}
	return Request{
		Method:  requestLineArr[0],
		URL:     requestLineArr[1],
		Proto:   requestLineArr[2],
		headers: headers,
	}
}

func router(req Request, conn net.Conn) {
	defer conn.Close()
	headers := make(map[string]string)
	switch true {
	case req.URL == "/":
		if _, err := conn.Write(constructResponse(200, "OK", nil, nil)); err != nil {
			fmt.Println("error writing to connection", err.Error())
		}
	case strings.HasPrefix(req.URL, "/echo"):
		echoStr := strings.TrimPrefix(req.URL, "/echo/")
		headers["Content-Type"] = "text/plain"
		headers["Content-Length"] = strconv.Itoa(len(echoStr))
		if _, err := conn.Write(constructResponse(200, "OK", headers, &echoStr)); err != nil {
			fmt.Println("error writing to connection", err.Error())
		}
	case req.URL == "/user-agent":
		headers["Content-Type"] = "text/plain"
		userAgent := req.headers["user-agent"]
		headers["Content-Length"] = strconv.Itoa(len(userAgent))
		if _, err := conn.Write(constructResponse(200, "OK", headers, &userAgent)); err != nil {
			fmt.Println("error writing to connection", err.Error())
		}
	default:
		if _, err := conn.Write(constructResponse(404, "Not Found", nil, nil)); err != nil {
			fmt.Println("error writing to connection", err.Error())
		}

	}
}

func constructResponse(status int, message string, headers map[string]string, body *string) []byte {
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", status, message)
	fullMessage := statusLine

	for k, v := range headers {
		fullMessage += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	fullMessage += "\r\n"

	if body != nil {
		fullMessage += fmt.Sprintf("%s", *body)
	}
	return []byte(fullMessage)
}
