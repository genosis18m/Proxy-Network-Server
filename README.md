# Custom Network Proxy Server

A forward proxy server implementation in Go that handles HTTP and HTTPS traffic over TCP. This project demonstrates fundamental systems and networking concepts including socket programming, concurrency, request parsing, forwarding logic, logging, and rule-based filtering.

## Features

- ✅ **TCP-based Proxy**: Raw socket implementation using Go's `net` package
- ✅ **HTTP Forwarding**: Parse and forward HTTP requests with URI cleaning
- ✅ **HTTPS Tunneling**: CONNECT method support for encrypted traffic
- ✅ **Domain Blocking**: Configurable blocklist with subdomain matching
- ✅ **Concurrent Handling**: Goroutine-per-connection architecture
- ✅ **Request Logging**: Detailed logging of all proxy activity
- ✅ **Configurable**: JSON-based configuration

## Project Structure

```
Proxy_Connection_ACM/
├── main.go                 # Entry point, config loading, TCP listener
├── proxy/
│   └── handler.go          # Request handling, tunneling, blocklist
├── config.json             # Server configuration
├── blocked_domains.txt     # Domain blocklist
├── test_proxy.sh           # Automated test script
├── Makefile                # Build system
├── DESIGN.md               # Architecture documentation
├── go.mod                  # Go module (1.21+)
└── proxy.log               # Runtime logs (created at runtime)
```

## Quick Start

### 1. Build the Proxy

```bash
make build
```

### 2. Run the Proxy Server

```bash
make run
```

You should see:
```
2026/01/08 18:14:04 Loaded 3 blocked domains
2026/01/08 18:14:04 Proxy server listening on :8080
```

### 3. Test the Proxy (in another terminal)

```bash
make test
```

## Usage Examples

### Using curl with the Proxy

```bash
# HTTP request through proxy
curl -x http://localhost:8080 http://httpbin.org/get

# HTTPS request through proxy (uses CONNECT tunneling)
curl -x http://localhost:8080 https://httpbin.org/get

# HEAD request
curl -I -x http://localhost:8080 http://httpbin.org/get

# Test blocked site (returns 403 Forbidden)
curl -x http://localhost:8080 http://example.com
```

### Expected Output - Allowed Site

```bash
$ curl -x http://localhost:8080 http://httpbin.org/get
{
  "args": {}, 
  "headers": {
    "Accept": "*/*", 
    "Host": "httpbin.org", 
    "User-Agent": "curl/8.5.0"
  }, 
  "origin": "103.37.201.190", 
  "url": "http://httpbin.org/get"
}
```

### Expected Output - Blocked Site

```bash
$ curl -x http://localhost:8080 http://example.com
403 Forbidden
```

## Configuring Your Browser

### Linux (Ubuntu) - System Proxy

1. Open **Settings** → **Network** → **Network Proxy**
2. Change from "Automatic" to **Manual**
3. Set HTTP Proxy: `localhost` Port: `8080`
4. Set HTTPS Proxy: `localhost` Port: `8080`
5. Click **Apply system-wide** (if available)

### Chrome / Edge (Linux)

Chrome uses system proxy settings on Linux. Configure as above.

### Firefox

1. Open **Settings** → **Network Settings**
2. Select **Manual proxy configuration**
3. HTTP Proxy: `localhost` Port: `8080`
4. Check "Also use this proxy for HTTPS"

## Configuration

### config.json

```json
{
  "port": 8080,
  "log_path": "proxy.log",
  "blocked_file_path": "blocked_domains.txt"
}
```

| Field | Description |
|-------|-------------|
| `port` | TCP port for proxy server |
| `log_path` | Path to log file |
| `blocked_file_path` | Path to blocked domains file |

### blocked_domains.txt

```
example.com
badsite.org
facebook.com
# Comments start with #
```

> **Note**: Restart the proxy after modifying the blocklist.

## Sample Log Output

```
[2026-01-08 18:10:00] Client: [::1]:52174 | Host: httpbin.org | Method: GET | Status: OK | Details: /get
[2026-01-08 18:10:01] Client: [::1]:52182 | Host: httpbin.org | Method: CONNECT | Status: OK | Details: Tunnel established
[2026-01-08 18:10:03] Client: [::1]:52198 | Host: example.com | Method: GET | Status: BLOCKED | Details: Domain is blocked
[2026-01-08 18:29:34] Client: [::1]:50578 | Host: www.facebook.com | Method: CONNECT | Status: BLOCKED | Details: Domain is blocked
```

## Test Results

The test script validates:

| Test | Description | Expected |
|------|-------------|----------|
| 1 | HTTP GET to httpbin.org | 200 OK |
| 2 | HTTPS GET to httpbin.org | 200 OK |
| 3 | HTTP to blocked example.com | 403 Forbidden |
| 4 | HTTPS to blocked example.com | Connection rejected |
| 5 | HTTP to blocked badsite.org | 403 Forbidden |
| 6 | Concurrent clients (10 parallel) | All succeed |
| 7 | Malformed request | 400 Bad Request |
| 8 | HEAD request | 200 OK |

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Compile the proxy server |
| `make run` | Build and run the server |
| `make test` | Run automated test script |
| `make clean` | Remove build artifacts |
| `make fmt` | Format Go source files |
| `make vet` | Run Go vet |

## Technical Details

- **Language**: Go 1.21+
- **Architecture**: TCP listener with goroutine-per-connection
- **Synchronization**: `sync.WaitGroup` for bidirectional copy
- **Timeout**: 10 seconds for target connections
- **Blocklist**: O(1) lookup with subdomain matching

## Documentation

See [DESIGN.md](DESIGN.md) for:
- High-level architecture diagrams
- Concurrency model explanation
- Data flow for HTTP and HTTPS
- Error handling strategies
- Security considerations
- Limitations

## License

This project is for educational purposes.
