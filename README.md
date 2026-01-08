# Custom Network Proxy Server

A forward proxy server in Go that handles HTTP and HTTPS traffic. Implements socket programming, concurrency, request parsing, and domain filtering.

## Features

- TCP-based proxy using Go's `net` package
- HTTP request forwarding with URI cleaning
- HTTPS tunneling via CONNECT method
- Domain blocking with subdomain matching
- Goroutine-per-connection model
- Request logging

## Building and Running

```bash
# Build
make build

# Run
make run

# Test
make test
```

## Usage

```bash
# HTTP request
curl -x http://localhost:8080 http://httpbin.org/get

# HTTPS request
curl -x http://localhost:8080 https://httpbin.org/get

# Blocked site (returns 403)
curl -x http://localhost:8080 http://example.com
```

## Configuration

Edit `config.json`:
```json
{
  "port": 8080,
  "log_path": "proxy.log",
  "blocked_file_path": "blocked_domains.txt"
}
```

Add domains to block in `blocked_domains.txt` (one per line).

## Browser Setup (Linux)

Go to Settings > Network > Network Proxy, set to Manual, and enter `localhost:8080` for HTTP and HTTPS proxy.

## Log Format

```
[2026-01-08 18:10:00] Client: 127.0.0.1:52174 | Host: httpbin.org | Method: GET | Status: OK
```

## Project Structure

```
├── main.go              # Entry point
├── proxy/handler.go     # Request handling
├── config.json          # Configuration
├── blocked_domains.txt  # Blocked domains
├── test_proxy.sh        # Tests
├── Makefile
└── DESIGN.md
```
