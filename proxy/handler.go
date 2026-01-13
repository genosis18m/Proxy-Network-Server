package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// HandleConnection processes an incoming proxy connection
func HandleConnection(clientConn net.Conn, blocklist map[string]bool, logFile *os.File) {
	defer clientConn.Close()

	// Read the first line of the request
	reader := bufio.NewReader(clientConn)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		logConnection(logFile, clientConn.RemoteAddr().String(), "", "", "ERROR", "Failed to read request")
		return
	}

	// Parse the request line (METHOD URI HTTP/VERSION)
	parts := strings.Fields(requestLine)
	if len(parts) < 3 {
		logConnection(logFile, clientConn.RemoteAddr().String(), "", "", "ERROR", "Invalid request line")
		sendError(clientConn, 400, "Bad Request")
		return
	}

	method := strings.ToUpper(parts[0])
	rawURI := parts[1]
	httpVersion := parts[2]

	// Read remaining headers
	headers := make([]string, 0)
	var hostHeader string
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" || line == "\n" {
			break
		}
		headers = append(headers, line)
		if strings.HasPrefix(strings.ToLower(line), "host:") {
			hostHeader = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "host:"))
			hostHeader = strings.TrimSpace(line[5:]) // Preserve original case
		}
	}

	// Determine target host and port
	var targetHost string
	var targetPort string

	if method == "CONNECT" {
		// CONNECT method: URI is host:port
		targetHost, targetPort = parseHostPort(rawURI, "443")
	} else {
		// HTTP methods: Parse from URI or Host header
		targetHost, targetPort = extractHostFromURI(rawURI, hostHeader)
	}

	// Check blocklist
	if isBlocked(targetHost, blocklist) {
		logConnection(logFile, clientConn.RemoteAddr().String(), targetHost, method, "BLOCKED", "Domain is blocked")
		sendError(clientConn, 403, "Forbidden")
		return
	}

	if method == "CONNECT" {
		handleHTTPS(clientConn, targetHost, targetPort, logFile)
	} else {
		handleHTTP(clientConn, reader, method, rawURI, httpVersion, headers, targetHost, targetPort, logFile)
	}
}

// handleHTTPS handles CONNECT requests for HTTPS tunneling
func handleHTTPS(clientConn net.Conn, targetHost, targetPort string, logFile *os.File) {
	targetAddr := net.JoinHostPort(targetHost, targetPort)

	// Connect to target server
	targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		logConnection(logFile, clientConn.RemoteAddr().String(), targetHost, "CONNECT", "ERROR", fmt.Sprintf("Failed to connect: %v", err))
		sendError(clientConn, 502, "Bad Gateway")
		return
	}
	defer targetConn.Close()

	// Send connection established response
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		logConnection(logFile, clientConn.RemoteAddr().String(), targetHost, "CONNECT", "ERROR", "Failed to send response")
		return
	}

	logConnection(logFile, clientConn.RemoteAddr().String(), targetHost, "CONNECT", "OK", "Tunnel established")

	// Bidirectional copy with WaitGroup for proper synchronization
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Target
	go func() {
		defer wg.Done()
		io.Copy(targetConn, clientConn)
		// Signal EOF to target
		if tcpConn, ok := targetConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// Target -> Client
	go func() {
		defer wg.Done()
		io.Copy(clientConn, targetConn)
		// Signal EOF to client
		if tcpConn, ok := clientConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// Wait for both directions to complete
	wg.Wait()
}

// handleHTTP handles regular HTTP requests
func handleHTTP(clientConn net.Conn, reader *bufio.Reader, method, rawURI, httpVersion string, headers []string, targetHost, targetPort string, logFile *os.File) {
	targetAddr := net.JoinHostPort(targetHost, targetPort)

	// Connect to target server
	targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		logConnection(logFile, clientConn.RemoteAddr().String(), targetHost, method, "ERROR", fmt.Sprintf("Failed to connect: %v", err))
		sendError(clientConn, 502, "Bad Gateway")
		return
	}
	defer targetConn.Close()

	// Clean the request URI (convert absolute to relative)
	cleanedURI := cleanRequestURI(rawURI)

	// Forward the request to target
	requestLine := fmt.Sprintf("%s %s %s\r\n", method, cleanedURI, httpVersion)
	_, err = targetConn.Write([]byte(requestLine))
	if err != nil {
		logConnection(logFile, clientConn.RemoteAddr().String(), targetHost, method, "ERROR", "Failed to forward request")
		sendError(clientConn, 502, "Bad Gateway")
		return
	}

	// Forward headers
	for _, header := range headers {
		targetConn.Write([]byte(header))
	}
	targetConn.Write([]byte("\r\n"))

	logConnection(logFile, clientConn.RemoteAddr().String(), targetHost, method, "OK", cleanedURI)

	// Bidirectional copy with WaitGroup for proper synchronization
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Target (forward any remaining body data)
	go func() {
		defer wg.Done()
		io.Copy(targetConn, reader)
		if tcpConn, ok := targetConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// Target -> Client
	go func() {
		defer wg.Done()
		io.Copy(clientConn, targetConn)
		if tcpConn, ok := clientConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// Wait for both directions to complete
	wg.Wait()
}

// parseHostPort splits host:port string, using defaultPort if not specified
func parseHostPort(addr, defaultPort string) (string, string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// No port specified
		return addr, defaultPort
	}
	return host, port
}

// extractHostFromURI extracts host and port from URI or Host header
func extractHostFromURI(rawURI, hostHeader string) (string, string) {
	// Try to parse as URL
	if strings.HasPrefix(rawURI, "http://") || strings.HasPrefix(rawURI, "https://") {
		if u, err := url.Parse(rawURI); err == nil {
			host := u.Hostname()
			port := u.Port()
			if port == "" {
				if u.Scheme == "https" {
					port = "443"
				} else {
					port = "80"
				}
			}
			return host, port
		}
	}

	// Fall back to Host header
	if hostHeader != "" {
		return parseHostPort(strings.TrimSpace(hostHeader), "80")
	}

	return "", "80"
}

// cleanRequestURI converts absolute URIs to relative paths
func cleanRequestURI(rawURI string) string {
	if strings.HasPrefix(rawURI, "http://") || strings.HasPrefix(rawURI, "https://") {
		if u, err := url.Parse(rawURI); err == nil {
			path := u.Path
			if path == "" {
				path = "/"
			}
			if u.RawQuery != "" {
				path += "?" + u.RawQuery
			}
			return path
		}
	}
	return rawURI
}

// isBlocked checks if the host is in the blocklist
func isBlocked(host string, blocklist map[string]bool) bool {
	// Normalize host (lowercase, strip port if present)
	host = strings.ToLower(host)
	// Use SplitHostPort only when host includes a port; if it errors, keep host unchanged
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host == "" {
		return false
	}

	// Check exact match
	if blocklist[host] {
		return true
	}

	// Check parent domains (e.g., sub.example.com should match example.com)
	parts := strings.Split(host, ".")
	for i := 1; i < len(parts); i++ {
		parentDomain := strings.Join(parts[i:], ".")
		if blocklist[parentDomain] {
			return true
		}
	}

	return false
}

// sendError sends an HTTP error response to the client
func sendError(conn net.Conn, statusCode int, statusText string) {
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n%d %s\n",
		statusCode, statusText, statusCode, statusText)
	conn.Write([]byte(response))
}

// logConnection logs connection details to the log file
func logConnection(logFile *os.File, clientAddr, targetHost, method, status, details string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] Client: %s | Host: %s | Method: %s | Status: %s | Details: %s\n",
		timestamp, clientAddr, targetHost, method, status, details)

	// Thread-safe write using file lock
	logFile.WriteString(logLine)
}
