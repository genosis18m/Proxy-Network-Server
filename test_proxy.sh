#!/bin/bash

# Test script for proxy server

PROXY_PORT=${1:-8080}
PROXY="http://localhost:${PROXY_PORT}"

echo "Testing proxy at ${PROXY}"
echo ""

# Test 1: HTTP
echo "Test 1: HTTP GET"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 10 -x ${PROXY} http://httpbin.org/get)
echo "httpbin.org -> ${STATUS}"

# Test 2: HTTPS
echo ""
echo "Test 2: HTTPS GET"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 10 -x ${PROXY} https://httpbin.org/get)
echo "httpbin.org (HTTPS) -> ${STATUS}"

# Test 3: Blocked site
echo ""
echo "Test 3: Blocked site"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 10 -x ${PROXY} http://example.com)
echo "example.com -> ${STATUS} (should be 403)"

# Test 4: Another blocked site
echo ""
echo "Test 4: Another blocked site"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 10 -x ${PROXY} http://badsite.org)
echo "badsite.org -> ${STATUS} (should be 403)"

# Test 5: HEAD request
echo ""
echo "Test 5: HEAD request"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' -I --connect-timeout 10 -x ${PROXY} http://httpbin.org/get)
echo "HEAD request -> ${STATUS}"

echo ""
echo "Done."
