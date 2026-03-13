# HTTP Server in Go

An HTTP/1.1 server built from scratch using only the Go standard library. No frameworks, no third-party dependencies, just raw TCP sockets and the `net` package.

Built as part of the [CodeCrafters "Build Your Own HTTP Server"](https://app.codecrafters.io/courses/http-server/overview) challenge.

## Features

- **Request parsing**: parses HTTP/1.1 request lines, headers, and body from raw TCP streams
- **Routing**: pattern-matched routing for multiple endpoints
- **File serving**: read and write files from a configurable directory
- **Gzip compression**: content negotiation via `Accept-Encoding`, with gzip support
- **Persistent connections**: HTTP/1.1 keep-alive with proper `Connection: close` handling
- **Concurrent clients** : goroutine-per-connection model

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Returns `200 OK` |
| GET | `/echo/{text}` | Echoes back `{text}` as plain text (supports gzip) |
| GET | `/user-agent` | Returns the client's `User-Agent` header |
| GET | `/files/{filename}` | Returns file contents from the serving directory |
| POST | `/files/{filename}` | Creates a file with the request body |

## Usage

```sh
# Build
go build -o http-server ./app

# Run (default serves current directory)
./http-server

# Run with a custom file directory
./http-server -directory /tmp/files
```

## Examples

```sh
# Basic request
curl -v http://localhost:4221/

# Echo with gzip compression
curl -v --header "Accept-Encoding: gzip" http://localhost:4221/echo/hello

# User agent
curl -v http://localhost:4221/user-agent

# Upload a file
curl -v --data "file contents here" http://localhost:4221/files/test.txt

# Download a file
curl -v http://localhost:4221/files/test.txt
```

## Running tests

```sh
go test -v ./app
```

## What I learned

- How HTTP/1.1 works at the TCP level — request/response framing, CRLF delimiters, `Content-Length` semantics
- Implementing content negotiation and gzip compression from scratch
- Managing persistent connections and knowing when to close them
- Parsing structured text protocols with `bufio`
