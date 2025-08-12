#!/bin/bash

# Test Coverage Script for Viva Rate Limiter
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Running Test Coverage Analysis for Viva Rate Limiter${NC}"
echo "=================================================="

# Create coverage directory if it doesn't exist
mkdir -p coverage

# Clean previous coverage data
rm -f coverage/coverage.out coverage/coverage.html coverage/coverage.json

echo -e "\n${YELLOW}📊 Running tests with coverage...${NC}"

# Run tests with coverage, excluding problematic packages
go test \
    -coverprofile=coverage/coverage.out \
    -covermode=atomic \
    -coverpkg=./internal/...,./pkg/... \
    $(go list ./... | grep -v '/scripts' | grep -v '/queue' | grep -v '/worker')

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Tests passed successfully${NC}"
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi

# Generate HTML coverage report
echo -e "\n${YELLOW}📈 Generating HTML coverage report...${NC}"
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Generate coverage summary
echo -e "\n${YELLOW}📊 Coverage Summary:${NC}"
go tool cover -func=coverage/coverage.out | tail -1

# Extract total coverage percentage
COVERAGE=$(go tool cover -func=coverage/coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')

echo -e "\n${BLUE}📋 Coverage Report Details:${NC}"
echo "==============================================="
echo "📁 HTML Report: coverage/coverage.html"
echo "📄 Raw Data:   coverage/coverage.out"
echo "🎯 Target:     50.0%"
echo "📊 Achieved:   ${COVERAGE}%"

# Check if we've met the target
if (( $(echo "$COVERAGE >= 50.0" | bc -l) )); then
    echo -e "${GREEN}🎉 Coverage target achieved! (${COVERAGE}% >= 50.0%)${NC}"
    
    # Generate badge-friendly JSON
    cat > coverage/coverage.json << EOF
{
  "schemaVersion": 1,
  "label": "coverage",
  "message": "${COVERAGE}%",
  "color": "brightgreen"
}
EOF
else
    echo -e "${YELLOW}⚠️  Coverage below target (${COVERAGE}% < 50.0%)${NC}"
    
    # Generate badge-friendly JSON
    cat > coverage/coverage.json << EOF
{
  "schemaVersion": 1,
  "label": "coverage",
  "message": "${COVERAGE}%",
  "color": "yellow"
}
EOF
fi

# Show top functions by coverage
echo -e "\n${BLUE}🔍 Coverage by Package:${NC}"
echo "========================================"
go tool cover -func=coverage/coverage.out | grep -E "^[^[:space:]]" | head -10

# Show uncovered functions (for improvement)
echo -e "\n${BLUE}⚠️  Functions needing test coverage:${NC}"
echo "========================================"
go tool cover -func=coverage/coverage.out | grep "0.0%" | head -10 | awk '{print $1 ":" $2}'

echo -e "\n${GREEN}✨ Coverage analysis complete!${NC}"
echo -e "📖 Open coverage/coverage.html in your browser to view detailed coverage report"

# If running in CI, also output coverage for badges/reporting
if [ "$CI" = "true" ]; then
    echo "COVERAGE_PERCENTAGE=${COVERAGE}" >> $GITHUB_ENV
    echo -e "\n${BLUE}🔧 CI: Coverage percentage set to ${COVERAGE}%${NC}"
fi