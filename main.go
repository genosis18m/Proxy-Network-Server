package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"proxy-server/proxy"
)

// Config represents the proxy server configuration
type Config struct {
	Port             int    `json:"port"`
	LogPath          string `json:"log_path"`
	BlockedFilePath  string `json:"blocked_file_path"`
}

func main() {
	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Load blocked domains
	blocklist, err := loadBlockedDomains(config.BlockedFilePath)
	if err != nil {
		log.Fatalf("Failed to load blocked domains: %v", err)
	}
	log.Printf("Loaded %d blocked domains", len(blocklist))

	// Open log file
	logFile, err := os.OpenFile(config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Start TCP listener
	addr := fmt.Sprintf(":%d", config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Proxy server listening on %s", addr)

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each connection in a goroutine
		go proxy.HandleConnection(conn, blocklist, logFile)
	}
}

// loadConfig reads and parses the configuration file
func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("unable to parse config file: %w", err)
	}

	return &config, nil
}

// loadBlockedDomains reads the blocked domains file and returns a map for O(1) lookup
func loadBlockedDomains(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open blocked domains file: %w", err)
	}
	defer file.Close()

	blocklist := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain != "" && !strings.HasPrefix(domain, "#") {
			blocklist[strings.ToLower(domain)] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading blocked domains: %w", err)
	}

	return blocklist, nil
}
