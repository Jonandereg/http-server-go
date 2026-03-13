package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestParseHTTPRequest_GET(t *testing.T) {
	raw := []byte("GET /echo/hello HTTP/1.1\r\nHost: localhost\r\nUser-Agent: test-client\r\n\r\n")
	req, err := parseHTTPRequest(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("Method = %q, want %q", req.Method, "GET")
	}
	if req.URL != "/echo/hello" {
		t.Errorf("URL = %q, want %q", req.URL, "/echo/hello")
	}
	if req.Proto != "HTTP/1.1" {
		t.Errorf("Proto = %q, want %q", req.Proto, "HTTP/1.1")
	}
	if req.headers["host"] != "localhost" {
		t.Errorf("Host header = %q, want %q", req.headers["host"], "localhost")
	}
	if req.headers["user-agent"] != "test-client" {
		t.Errorf("User-Agent header = %q, want %q", req.headers["user-agent"], "test-client")
	}
}

func TestParseHTTPRequest_POST_WithBody(t *testing.T) {
	raw := []byte("POST /files/test.txt HTTP/1.1\r\nContent-Length: 13\r\n\r\nHello, World!")
	req, err := parseHTTPRequest(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("Method = %q, want %q", req.Method, "POST")
	}
	if string(req.body) != "Hello, World!" {
		t.Errorf("Body = %q, want %q", string(req.body), "Hello, World!")
	}
}

func TestParseHTTPRequest_HeadersCaseInsensitive(t *testing.T) {
	raw := []byte("GET / HTTP/1.1\r\nContent-Type: text/plain\r\nACCEPT-ENCODING: gzip\r\n\r\n")
	req, err := parseHTTPRequest(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.headers["content-type"] != "text/plain" {
		t.Errorf("content-type = %q, want %q", req.headers["content-type"], "text/plain")
	}
	if req.headers["accept-encoding"] != "gzip" {
		t.Errorf("accept-encoding = %q, want %q", req.headers["accept-encoding"], "gzip")
	}
}

func TestConstructResponse_NoBody(t *testing.T) {
	resp := constructResponse(200, "OK", nil, nil)
	expected := "HTTP/1.1 200 OK\r\n\r\n"
	if string(resp) != expected {
		t.Errorf("Response = %q, want %q", string(resp), expected)
	}
}

func TestConstructResponse_WithBody(t *testing.T) {
	headers := map[string]string{
		"Content-Type":   "text/plain",
		"Content-Length": "5",
	}
	body := "hello"
	resp := constructResponse(200, "OK", headers, &body)
	got := string(resp)

	// Check status line
	if !bytes.HasPrefix(resp, []byte("HTTP/1.1 200 OK\r\n")) {
		t.Errorf("Response missing status line, got: %q", got)
	}
	// Check body at the end
	if !bytes.HasSuffix(resp, []byte("\r\nhello")) {
		t.Errorf("Response missing body, got: %q", got)
	}
	// Check headers present
	if !bytes.Contains(resp, []byte("Content-Type: text/plain\r\n")) {
		t.Errorf("Response missing Content-Type header, got: %q", got)
	}
	if !bytes.Contains(resp, []byte("Content-Length: 5\r\n")) {
		t.Errorf("Response missing Content-Length header, got: %q", got)
	}
}

func TestConstructResponse_404(t *testing.T) {
	resp := constructResponse(404, "Not Found", nil, nil)
	expected := "HTTP/1.1 404 Not Found\r\n\r\n"
	if string(resp) != expected {
		t.Errorf("Response = %q, want %q", string(resp), expected)
	}
}

func TestDetectEncoding_Gzip(t *testing.T) {
	headers := map[string]string{
		"accept-encoding": "gzip",
	}
	enc := detectEncoding(headers)
	if enc != "gzip" {
		t.Errorf("Encoding = %q, want %q", enc, "gzip")
	}
}

func TestDetectEncoding_MultipleWithGzip(t *testing.T) {
	headers := map[string]string{
		"accept-encoding": "deflate, gzip, br",
	}
	enc := detectEncoding(headers)
	if enc != "gzip" {
		t.Errorf("Encoding = %q, want %q", enc, "gzip")
	}
}

func TestDetectEncoding_Unsupported(t *testing.T) {
	headers := map[string]string{
		"accept-encoding": "br, deflate",
	}
	enc := detectEncoding(headers)
	if enc != "" {
		t.Errorf("Encoding = %q, want empty string", enc)
	}
}

func TestDetectEncoding_NoHeader(t *testing.T) {
	headers := map[string]string{}
	enc := detectEncoding(headers)
	if enc != "" {
		t.Errorf("Encoding = %q, want empty string", enc)
	}
}

func TestCompressBody_Gzip(t *testing.T) {
	input := []byte("hello world")
	compressed, err := compressBody(input, "gzip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(compressed) == 0 {
		t.Fatal("compressed body is empty")
	}

	// Decompress and verify
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer reader.Close()
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}
	if string(decompressed) != "hello world" {
		t.Errorf("Decompressed = %q, want %q", string(decompressed), "hello world")
	}
}

func TestCompressBody_UnsupportedType(t *testing.T) {
	_, err := compressBody([]byte("hello"), "deflate")
	if err == nil {
		t.Fatal("expected error for unsupported compression type")
	}
}
