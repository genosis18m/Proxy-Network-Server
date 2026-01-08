#!/bin/bash

# Test script for Custom Network Proxy Server
# Usage: ./test_proxy.sh [proxy_port]

PROXY_PORT=${1:-8080}
PROXY_URL="http://localhost:${PROXY_PORT}"

echo "=============================================="
echo "Custom Network Proxy Server - Test Suite"
echo "=============================================="
echo "Proxy URL: ${PROXY_URL}"
echo ""

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_status="$3"
    
    echo -n "Test: ${test_name}... "
    
    # Execute the command and capture HTTP status code
    HTTP_STATUS=$(eval "${test_command}" 2>/dev/null)
    
    if [ "$HTTP_STATUS" == "$expected_status" ]; then
        echo -e "${GREEN}PASSED${NC} (Status: ${HTTP_STATUS})"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}FAILED${NC} (Expected: ${expected_status}, Got: ${HTTP_STATUS})"
        ((TESTS_FAILED++))
    fi
}

echo "----------------------------------------------"
echo "Test 1: Valid HTTP Site (httpbin.org)"
echo "----------------------------------------------"
echo "Command: curl -s -o /dev/null -w '%{http_code}' -x ${PROXY_URL} http://httpbin.org/get"
run_test "HTTP GET to httpbin.org" \
    "curl -s -o /dev/null -w '%{http_code}' --connect-timeout 10 -x ${PROXY_URL} http://httpbin.org/get" \
    "200"
echo ""

echo "----------------------------------------------"
echo "Test 2: Valid HTTPS Site (httpbin.org)"
echo "----------------------------------------------"
echo "Command: curl -s -o /dev/null -w '%{http_code}' -x ${PROXY_URL} https://httpbin.org/get"
run_test "HTTPS GET to httpbin.org" \
    "curl -s -o /dev/null -w '%{http_code}' --connect-timeout 10 -x ${PROXY_URL} https://httpbin.org/get" \
    "200"
echo ""

echo "----------------------------------------------"
echo "Test 3: Blocked Site (example.com)"
echo "----------------------------------------------"
echo "Command: curl -s -o /dev/null -w '%{http_code}' -x ${PROXY_URL} http://example.com"
run_test "HTTP GET to blocked example.com" \
    "curl -s -o /dev/null -w '%{http_code}' --connect-timeout 10 -x ${PROXY_URL} http://example.com" \
    "403"
echo ""

echo "----------------------------------------------"
echo "Test 4: Blocked Site HTTPS (example.com)"
echo "----------------------------------------------"
echo "Command: curl -s -o /dev/null -w '%{http_code}' -x ${PROXY_URL} https://example.com"
# Note: When CONNECT is rejected, curl may return 000 (connection failed) or 403
HTTP_STATUS=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 10 -x ${PROXY_URL} https://example.com 2>/dev/null)
echo -n "Test: HTTPS CONNECT to blocked example.com... "
if [ "$HTTP_STATUS" == "403" ] || [ "$HTTP_STATUS" == "000" ]; then
    echo -e "${GREEN}PASSED${NC} (Status: ${HTTP_STATUS} - connection rejected)"
    ((TESTS_PASSED++))
else
    echo -e "${RED}FAILED${NC} (Expected: 403 or 000, Got: ${HTTP_STATUS})"
    ((TESTS_FAILED++))
fi
echo ""

echo "----------------------------------------------"
echo "Test 5: Another Blocked Site (badsite.org)"
echo "----------------------------------------------"
echo "Command: curl -s -o /dev/null -w '%{http_code}' -x ${PROXY_URL} http://badsite.org"
run_test "HTTP GET to blocked badsite.org" \
    "curl -s -o /dev/null -w '%{http_code}' --connect-timeout 10 -x ${PROXY_URL} http://badsite.org" \
    "403"
echo ""

echo "=============================================="
echo "Test Results"
echo "=============================================="
echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
