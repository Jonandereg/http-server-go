package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Request struct {
	Method  string
	URL     string
	Proto   string
	headers map[string]string
	body    []byte
}

var supportedEncodings = map[string]bool{
	"gzip": true,
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	fmt.Println("Listening on 0.0.0.0:4221")
	dir := flag.String("directory", ".", "directory to serve")
	flag.Parse()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn, dir)
	}

}

func handleConnection(conn net.Conn, dir *string) {
	defer conn.Close()
	for {
		req := make([]byte, 1024)
		if _, err := conn.Read(req); err != nil {
			if err != io.EOF {
				fmt.Println("Error reading request: ", err.Error())
				respondServerError(conn)
			}
			return
		}
		fmt.Println("Request received: ", string(req))

		parsedReq, err := parseHTTPRequest(req)
		if err != nil {
			fmt.Println("Error parsing request: ", err.Error())
			respondServerError(conn)
		}

		shouldClose := router(parsedReq, conn, dir)
		if shouldClose {
			return
		}
	}
}

func parseHTTPRequest(rawReq []byte) (Request, error) {
	r := bufio.NewReader(bytes.NewReader(rawReq))

	requestLine, err := r.ReadString('\n')
	if err != nil {
		return Request{}, fmt.Errorf("error reading request: %q", err.Error())

	}
	requestLineArr := strings.Split(strings.Trim(requestLine, "\r\n"), " ")

	headers := make(map[string]string)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return Request{}, fmt.Errorf("error reading request: %q", err.Error())
		}
		if line == "\r\n" {
			break
		}
		trimmedLine := strings.Trim(line, "\r\n")
		keyValue := strings.Split(trimmedLine, ":")
		if len(keyValue) != 2 {
			continue
		}
		headers[strings.TrimSpace(strings.ToLower(keyValue[0]))] = strings.TrimSpace(keyValue[1])
	}
	var body []byte
	contentLengthStr, exist := headers["content-length"]
	if exist {
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			fmt.Println("Error parsing Content-Length: ", err.Error())
			os.Exit(1)
		}
		body = make([]byte, contentLength)
		if _, err := io.ReadFull(r, body); err != nil {
			return Request{}, fmt.Errorf("error reading request body: %q", err.Error())
		}
	}

	return Request{
		Method:  requestLineArr[0],
		URL:     requestLineArr[1],
		Proto:   requestLineArr[2],
		headers: headers,
		body:    body,
	}, nil
}

func router(req Request, conn net.Conn, dir *string) bool {
	headers := make(map[string]string)
	closeConnection, exists := req.headers["connection"]
	var shouldClose bool
	if exists && closeConnection == "close" {
		headers["Connection"] = "close"
		shouldClose = true
	}

	switch {
	case req.URL == "/":
		if _, err := conn.Write(constructResponse(200, "OK", nil, nil)); err != nil {
			fmt.Println("error writing to connection", err.Error())
		}
	case strings.HasPrefix(req.URL, "/echo"):
		echoStr := strings.TrimPrefix(req.URL, "/echo/")
		headers["Content-Type"] = "text/plain"
		encoding := detectEncoding(req.headers)
		if encoding != "" {
			headers["Content-Encoding"] = encoding
			body, err := compressBody([]byte(echoStr), encoding)
			if err != nil {
				fmt.Println("error compressing body:", err.Error())
				respondServerError(conn)
			}
			echoStr = string(body)
		}
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
	case strings.HasPrefix(req.URL, "/files"):
		filename := strings.TrimPrefix(req.URL, "/files/")
		if req.Method == "GET" {
			files, err := os.ReadDir(*dir)
			if err != nil {
				fmt.Println("Error reading directory: ", err.Error())
				respondServerError(conn)
			}
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				if file.Name() == filename {
					file, err := os.ReadFile(filepath.Join(*dir, filename))
					if err != nil {
						fmt.Println("Error reading file: ", err.Error())
						respondServerError(conn)
					}
					headers["Content-Type"] = "application/octet-stream"
					headers["Content-Length"] = strconv.Itoa(len(file))
					if _, err := conn.Write(constructResponse(200, "OK", headers, new(string(file)))); err != nil {
						fmt.Println("error writing to connection", err.Error())
					}
				}

			}

			respondNotFound(conn)
		}
		if req.Method == "POST" {
			if err := os.WriteFile(filepath.Join(*dir, filename), []byte(req.body), 0644); err != nil {
				fmt.Println("Error writing to file: ", err.Error())
				respondServerError(conn)
			}
			if _, err := conn.Write(constructResponse(201, "Created", nil, nil)); err != nil {
				fmt.Println("error writing to connection", err.Error())
			}
		}

	default:
		respondNotFound(conn)
	}
	return shouldClose
}

func respondNotFound(conn net.Conn) {
	if _, err := conn.Write(constructResponse(404, "Not Found", nil, nil)); err != nil {
		fmt.Println("error writing to connection", err.Error())
	}
}

func respondServerError(conn net.Conn) {
	if _, err := conn.Write(constructResponse(500, "Internal Server Error", nil, nil)); err != nil {
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

func detectEncoding(headers map[string]string) string {
	encoding := ""
	clientEncodingsStr, exists := headers["accept-encoding"]
	if !exists {
		return encoding
	}
	clientEncodings := strings.Split(clientEncodingsStr, ", ")
	for _, clientEncoding := range clientEncodings {
		if supportedEncodings[clientEncoding] {
			encoding = clientEncoding
			break
		}

	}
	return encoding
}

func compressBody(body []byte, compressionType string) (out []byte, err error) {
	switch compressionType {
	case "gzip":
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		defer func() {
			if cErr := gz.Close(); cErr != nil && err == nil {
				err = errors.New("Error closing gzip writer: " + err.Error())
			}
			out = buf.Bytes()
		}()
		if _, err := gz.Write(body); err != nil {

			return nil, err
		}
		return
	default:
		return nil, errors.New("unsupported compression type: " + compressionType)
	}
}
