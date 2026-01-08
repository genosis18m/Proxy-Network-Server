# Custom Network Proxy Server - Design Document

## Overview

This is a forward proxy server in Go that handles HTTP and HTTPS traffic. It uses raw TCP sockets and a goroutine-per-connection concurrency model.

## Architecture

```
Client  -->  Proxy Server (TCP :8080)  -->  Target Server
                    |
            [Parse Request]
                    |
            [Check Blocklist]
                    |
         [Forward/Tunnel Data]
```

The proxy accepts connections, parses the HTTP request to get the destination, checks if it's blocked, and either returns 403 or forwards the traffic.

## Concurrency Model

I chose goroutine-per-connection because:
- Simple to implement and understand
- Go handles the scheduling efficiently
- Each connection is independent

For each connection, a new goroutine is spawned. For bidirectional data transfer (especially HTTPS tunneling), two more goroutines handle the data flow in each direction. A WaitGroup ensures the handler doesn't exit until both directions finish.

```go
var wg sync.WaitGroup
wg.Add(2)

go func() {
    defer wg.Done()
    io.Copy(targetConn, clientConn)
}()

go func() {
    defer wg.Done()
    io.Copy(clientConn, targetConn)
}()

wg.Wait()
```

## Data Flow

### HTTP Requests

1. Client sends `GET http://site.com/path HTTP/1.1`
2. Proxy parses host and path
3. Proxy connects to site.com:80
4. Proxy cleans URI (absolute -> relative) and forwards request
5. Proxy relays response back to client

### HTTPS Requests (CONNECT)

1. Client sends `CONNECT site.com:443 HTTP/1.1`
2. Proxy connects to site.com:443
3. Proxy sends `HTTP/1.1 200 Connection Established`
4. Proxy tunnels encrypted bytes bidirectionally (doesn't inspect TLS)

## Blocklist

Domains are loaded from a text file into a map for O(1) lookup. The check includes subdomain matching - blocking `example.com` also blocks `www.example.com`.

When blocked, the proxy returns:
```
HTTP/1.1 403 Forbidden
```

## Logging

Each request is logged with timestamp, client IP, host, method, and status.

Example:
```
[2026-01-08 12:30:45] Client: 127.0.0.1:54321 | Host: httpbin.org | Method: GET | Status: OK
```

## Error Handling

- Connection timeouts: 10 second timeout on target connections
- Failed connections: Returns 502 Bad Gateway
- Invalid requests: Returns 400 Bad Request
- All errors are logged

## Limitations

- No persistent connections (connection closed after each request)
- Blocklist requires restart to reload
- No authentication
- No caching
- No rate limiting

## Security Notes

This is an educational project. For production use, you'd want to add:
- Authentication
- Rate limiting
- TLS for the proxy itself
- Better input validation
