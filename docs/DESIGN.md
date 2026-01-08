# Custom Network Proxy Server - Design Document

## Overview

This document describes the design and architecture of a Custom Network Proxy Server implemented in Golang. The proxy server handles both HTTP and HTTPS (via CONNECT tunneling) requests, supports domain-based blocklisting, and uses a goroutine-per-connection concurrency model.

## Table of Contents

1. [High-Level Architecture](#high-level-architecture)
2. [Concurrency Model](#concurrency-model)
3. [Data Flow](#data-flow)
4. [Component Details](#component-details)
5. [Blocklist Mechanism](#blocklist-mechanism)
6. [Logging Strategy](#logging-strategy)
7. [Configuration](#configuration)

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           CLIENT                                     │
│                    (Browser / curl / Application)                    │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
                                  │ HTTP Request / CONNECT Request
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       PROXY SERVER                                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                     TCP Listener (:8080)                       │  │
│  │                    net.Listen("tcp", addr)                     │  │
│  └───────────────────────────────┬───────────────────────────────┘  │
│                                  │                                   │
│                                  │ Accept()                          │
│                                  ▼                                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │           Connection Handler (goroutine per connection)        │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌───────────────────────┐ │  │
│  │  │ Parse HTTP  │─▶│ Check Block- │─▶│ Handle HTTP/HTTPS     │ │  │
│  │  │   Request   │  │    list      │  │ (Tunnel/Forward)      │ │  │
│  │  └─────────────┘  └──────────────┘  └───────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
                                  │ TCP Connection
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         TARGET SERVER                                │
│                    (example.com, httpbin.org, etc.)                  │
└─────────────────────────────────────────────────────────────────────┘
```

### Request Flow

1. **Client** sends an HTTP or HTTPS request to the proxy server
2. **Proxy Server** accepts the connection via TCP listener
3. **Handler Goroutine** is spawned for the connection
4. Request is parsed to extract method, host, and port
5. **Blocklist Check** determines if the domain is allowed
6. If blocked: return `403 Forbidden`
7. If allowed: establish connection to target and tunnel/forward data

---

## Concurrency Model

### Goroutine-per-Connection Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         MAIN GOROUTINE                               │
│                                                                      │
│   for {                                                              │
│       conn, _ := listener.Accept()                                   │
│       go proxy.HandleConnection(conn, blocklist, logFile)   ───────────┐
│   }                                                                  │ │
│                                                                      │ │
└──────────────────────────────────────────────────────────────────────┘ │
                                                                         │
    ┌────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      HANDLER GOROUTINE                               │
│                                                                      │
│   HandleConnection(conn, blocklist, logFile)                         │
│   ├── Parse request                                                  │
│   ├── Check blocklist                                                │
│   └── Handle connection (HTTP or HTTPS)                              │
│       │                                                              │
│       └── Spawn TWO child goroutines for bidirectional copy          │
│           │                                                          │
│           ├── Goroutine 1: Client → Target (io.Copy)                 │
│           │                                                          │
│           └── Goroutine 2: Target → Client (io.Copy)                 │
│                                                                      │
│       └── sync.WaitGroup.Wait() ← Block until BOTH complete          │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Synchronization with sync.WaitGroup

The bidirectional copy uses `sync.WaitGroup` to ensure:
- The handler does **not** return until **both** `io.Copy` directions complete
- No data is lost due to premature connection closure
- Proper cleanup of resources

```go
var wg sync.WaitGroup
wg.Add(2)

go func() {
    defer wg.Done()
    io.Copy(targetConn, clientConn)  // Client → Target
}()

go func() {
    defer wg.Done()
    io.Copy(clientConn, targetConn)  // Target → Client
}()

wg.Wait()  // Block until both complete
```

---

## Data Flow

### HTTP Request Flow

```
Client                          Proxy                           Target
  │                               │                               │
  │  GET http://site.com/path     │                               │
  │─────────────────────────────▶│                               │
  │                               │                               │
  │                               │ Parse request                 │
  │                               │ Extract host: site.com:80     │
  │                               │ Clean URI: /path              │
  │                               │                               │
  │                               │ Check blocklist → ALLOWED     │
  │                               │                               │
  │                               │     Connect to site.com:80    │
  │                               │──────────────────────────────▶│
  │                               │                               │
  │                               │     GET /path HTTP/1.1        │
  │                               │──────────────────────────────▶│
  │                               │                               │
  │                               │     HTTP/1.1 200 OK           │
  │                               │◀──────────────────────────────│
  │                               │                               │
  │      HTTP/1.1 200 OK          │                               │
  │◀──────────────────────────────│                               │
  │                               │                               │
```

### HTTPS (CONNECT) Request Flow

```
Client                          Proxy                           Target
  │                               │                               │
  │  CONNECT site.com:443         │                               │
  │─────────────────────────────▶│                               │
  │                               │                               │
  │                               │ Parse CONNECT request         │
  │                               │ Extract host: site.com:443    │
  │                               │                               │
  │                               │ Check blocklist → ALLOWED     │
  │                               │                               │
  │                               │     Connect to site.com:443   │
  │                               │──────────────────────────────▶│
  │                               │                               │
  │  HTTP/1.1 200 Connection      │                               │
  │           Established         │                               │
  │◀──────────────────────────────│                               │
  │                               │                               │
  │         TLS Handshake         │         TLS Handshake         │
  │◀═══════════════════════════════════════════════════════════▶ │
  │                               │                               │
  │       Encrypted Data          │       Encrypted Data          │
  │◀═══════════════════════════════════════════════════════════▶ │
  │                               │                               │
```

---

## Component Details

### main.go

| Component | Description |
|-----------|-------------|
| `Config` struct | Holds configuration: port, log path, blocked file path |
| `loadConfig()` | Reads and parses `config.json` |
| `loadBlockedDomains()` | Loads blocked domains into a map for O(1) lookup |
| Main loop | Accepts connections and spawns handler goroutines |

### proxy/handler.go

| Function | Description |
|----------|-------------|
| `HandleConnection()` | Main entry point for connection handling |
| `handleHTTPS()` | Handles CONNECT tunneling with WaitGroup sync |
| `handleHTTP()` | Handles HTTP forwarding with URI cleaning |
| `parseHostPort()` | Splits host:port, handles defaults |
| `extractHostFromURI()` | Extracts host from absolute URI or Host header |
| `cleanRequestURI()` | Converts absolute URIs to relative paths |
| `isBlocked()` | Checks domain against blocklist (including subdomains) |
| `sendError()` | Sends HTTP error responses |
| `logConnection()` | Logs connection details to file |

---

## Blocklist Mechanism

### Domain Matching

The blocklist supports:
1. **Exact match**: `example.com` blocks `example.com`
2. **Subdomain match**: `example.com` also blocks `sub.example.com`, `www.example.com`

```go
// Blocklist loaded as map[string]bool for O(1) lookup
blocklist["example.com"] = true
blocklist["badsite.org"] = true

// Checking includes parent domain traversal
// www.example.com → example.com → com
```

### Response

When a domain is blocked, the proxy returns:
```
HTTP/1.1 403 Forbidden
Content-Type: text/plain
Connection: close

403 Forbidden
```

---

## Logging Strategy

### Log Format

```
[TIMESTAMP] Client: IP:PORT | Host: DOMAIN | Method: METHOD | Status: STATUS | Details: INFO
```

### Example Log Entries

```
[2024-01-08 12:30:45] Client: 127.0.0.1:54321 | Host: httpbin.org | Method: GET | Status: OK | Details: /get
[2024-01-08 12:30:46] Client: 127.0.0.1:54322 | Host: example.com | Method: CONNECT | Status: BLOCKED | Details: Domain is blocked
[2024-01-08 12:30:47] Client: 127.0.0.1:54323 | Host: google.com | Method: CONNECT | Status: OK | Details: Tunnel established
```

---

## Configuration

### config.json

```json
{
  "port": 8080,
  "log_path": "proxy.log",
  "blocked_file_path": "blocked_domains.txt"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `port` | int | TCP port for the proxy server |
| `log_path` | string | Path to the log file |
| `blocked_file_path` | string | Path to the blocked domains file |

### blocked_domains.txt

One domain per line:
```
example.com
badsite.org
# Comments start with #
```

---

## Building and Running

```bash
# Build
make build

# Run
make run

# Test
make test

# Clean
make clean
```

---

## Error Handling

### Connection Errors

| Error Type | Handling Strategy |
|------------|-------------------|
| Failed to connect to target | Return `502 Bad Gateway` to client, log error |
| Connection timeout | 10-second timeout on `net.DialTimeout`, return `502` |
| Read/Write errors | Close connection gracefully, log error |
| Invalid request format | Return `400 Bad Request` to client |

### Request Parsing Errors

```go
// Invalid request line handling
if len(parts) < 3 {
    logConnection(logFile, clientAddr, "", "", "ERROR", "Invalid request line")
    sendError(clientConn, 400, "Bad Request")
    return
}
```

### Graceful Connection Closure

- `defer clientConn.Close()` ensures client connection is always closed
- `defer targetConn.Close()` ensures target connection is always closed
- TCP half-close with `CloseWrite()` signals EOF without dropping data

---

## Limitations

### Current Implementation Limitations

| Limitation | Description |
|------------|-------------|
| No persistent connections | Each request creates a new connection (no HTTP/1.1 keep-alive) |
| No chunked encoding parsing | Chunked transfer is forwarded transparently without interpretation |
| No request body size limits | Large uploads are forwarded without size restrictions |
| No connection pooling | Target connections are not reused |
| Single log file | No log rotation implemented |
| Memory-resident blocklist | Blocklist is loaded at startup, requires restart to update |

### Not Implemented (Optional Extensions)

| Feature | Status |
|---------|--------|
| Response caching | Not implemented |
| Proxy authentication | Not implemented |
| Rate limiting | Not implemented |
| HTTPS inspection (MITM) | Not implemented (only tunneling) |

---

## Security Considerations

### Input Validation

1. **Request Line Parsing**: Validates HTTP request format before processing
2. **Host Header Sanitization**: Strips port, converts to lowercase before blocklist check
3. **Domain Canonicalization**: Normalizes domains (lowercase, trim whitespace)

### Potential Vulnerabilities and Mitigations

| Vulnerability | Mitigation |
|---------------|------------|
| Request smuggling | Single request per connection, no keep-alive |
| Log injection | Log entries use fixed format, special characters not escaped (potential issue) |
| DNS rebinding | Not mitigated - proxy trusts DNS resolution |
| Open proxy abuse | Bind to localhost only, or implement authentication |
| Resource exhaustion | Connection timeout (10s), but no rate limiting |

### Recommendations for Production Use

1. **Bind to localhost**: Don't expose proxy to public internet without authentication
2. **Implement authentication**: Add Basic or Digest authentication for authorized users
3. **Add rate limiting**: Prevent abuse by limiting connections per IP
4. **Log rotation**: Implement log rotation to prevent disk exhaustion
5. **TLS for proxy connection**: Consider TLS between client and proxy for sensitive environments

### Current Security Posture

```
[WARNING] This proxy is intended for educational/development use.
For production deployment, implement:
- Authentication
- Rate limiting  
- TLS encryption
- Log sanitization
- Access control lists
```

---

## Building and Running

```bash
# Build
make build

# Run
make run

# Test
make test

# Clean
make clean
```

---

## Future Enhancements

1. **Authentication**: Add proxy authentication support
2. **Rate Limiting**: Implement connection rate limiting per client
3. **HTTPS Inspection**: Man-in-the-middle certificate for HTTPS inspection
4. **Caching**: Add response caching for frequently accessed resources
5. **Metrics**: Prometheus metrics endpoint for monitoring
6. **Graceful Shutdown**: Handle SIGTERM for clean shutdown

