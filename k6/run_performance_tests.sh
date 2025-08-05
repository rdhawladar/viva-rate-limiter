#!/bin/bash

# Performance Test Runner for Viva Rate Limiter
# Runs comprehensive performance tests and generates reports

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_URL="http://localhost:8090"
RESULTS_DIR="./results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

echo -e "${BLUE}🚀 Viva Rate Limiter Performance Test Suite${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}❌ k6 is not installed. Please install k6 first:${NC}"
    echo "   macOS: brew install k6"
    echo "   Linux: sudo apt-get install k6"
    echo "   Windows: choco install k6"
    echo "   Or download from: https://k6.io/docs/getting-started/installation/"
    exit 1
fi

# Check if API is running
echo -e "${YELLOW}🔍 Checking if API is running...${NC}"
if curl -s "${API_URL}/health" > /dev/null; then
    echo -e "${GREEN}✅ API is running at ${API_URL}${NC}"
else
    echo -e "${RED}❌ API is not running at ${API_URL}${NC}"
    echo "Please start the API server first:"
    echo "   make run-api"
    echo "   or"
    echo "   go run cmd/api/main.go"
    exit 1
fi

# Create results directory
mkdir -p "${RESULTS_DIR}"

# Function to run a test and save results
run_test() {
    local test_name="$1"
    local test_file="$2"
    local description="$3"
    
    echo ""
    echo -e "${BLUE}📊 Running ${test_name}...${NC}"
    echo -e "${YELLOW}${description}${NC}"
    echo ""
    
    local output_file="${RESULTS_DIR}/${test_name}_${TIMESTAMP}.json"
    local summary_file="${RESULTS_DIR}/${test_name}_${TIMESTAMP}_summary.txt"
    
    # Run the test
    if k6 run --out json="${output_file}" "${test_file}" | tee "${summary_file}"; then
        echo -e "${GREEN}✅ ${test_name} completed successfully${NC}"
        echo -e "${GREEN}📄 Results saved to: ${output_file}${NC}"
        echo -e "${GREEN}📋 Summary saved to: ${summary_file}${NC}"
        return 0
    else
        echo -e "${RED}❌ ${test_name} failed${NC}"
        return 1
    fi
}

# Function to show test menu
show_menu() {
    echo -e "${BLUE}Select tests to run:${NC}"
    echo "1) Basic Rate Limiter Performance Test"
    echo "2) API CRUD Operations Test"
    echo "3) Redis Stress Test"
    echo "4) Rate Limiting Accuracy Test"
    echo "5) All Tests (Sequential)"
    echo "6) All Tests (Parallel - Advanced)"
    echo "0) Exit"
    echo ""
}

# Function to run all tests sequentially
run_all_sequential() {
    echo -e "${BLUE}🏃 Running all tests sequentially...${NC}"
    
    local failed_tests=()
    
    if ! run_test "basic_performance" "rate_limiter_basic.js" "Tests core rate limiting with increasing load"; then
        failed_tests+=("Basic Performance")
    fi
    
    sleep 10 # Cool down between tests
    
    if ! run_test "crud_operations" "api_crud_operations.js" "Tests all API endpoints with CRUD operations"; then
        failed_tests+=("CRUD Operations")
    fi
    
    sleep 10
    
    if ! run_test "redis_stress" "redis_stress_test.js" "Tests Redis backend under high concurrent load"; then
        failed_tests+=("Redis Stress")
    fi
    
    sleep 10
    
    if ! run_test "rate_limiting_accuracy" "rate_limiting_accuracy.js" "Verifies rate limit enforcement accuracy"; then
        failed_tests+=("Rate Limiting Accuracy")
    fi
    
    # Summary
    echo ""
    echo -e "${BLUE}📊 Test Suite Summary${NC}"
    echo -e "${BLUE}===================${NC}"
    
    if [ ${#failed_tests[@]} -eq 0 ]; then
        echo -e "${GREEN}✅ All tests passed successfully!${NC}"
    else
        echo -e "${RED}❌ Failed tests: ${failed_tests[*]}${NC}"
    fi
    
    echo -e "${YELLOW}📁 Results saved in: ${RESULTS_DIR}${NC}"
}

# Function to run all tests in parallel (advanced)
run_all_parallel() {
    echo -e "${BLUE}🏃‍♂️ Running all tests in parallel (Advanced)...${NC}"
    echo -e "${YELLOW}⚠️  This puts maximum stress on the system${NC}"
    echo ""
    
    # Start all tests in background
    run_test "basic_performance_parallel" "rate_limiter_basic.js" "Basic performance (parallel)" &
    pid1=$!
    
    sleep 5 # Stagger start times
    
    run_test "crud_operations_parallel" "api_crud_operations.js" "CRUD operations (parallel)" &
    pid2=$!
    
    sleep 5
    
    run_test "redis_stress_parallel" "redis_stress_test.js" "Redis stress (parallel)" &
    pid3=$!
    
    # Wait for all tests to complete
    echo -e "${YELLOW}⏳ Waiting for all parallel tests to complete...${NC}"
    
    wait $pid1
    result1=$?
    
    wait $pid2  
    result2=$?
    
    wait $pid3
    result3=$?
    
    # Summary
    echo ""
    echo -e "${BLUE}📊 Parallel Test Suite Summary${NC}"
    echo -e "${BLUE}==============================${NC}"
    
    if [ $result1 -eq 0 ]; then
        echo -e "${GREEN}✅ Basic Performance: PASSED${NC}"
    else
        echo -e "${RED}❌ Basic Performance: FAILED${NC}"
    fi
    
    if [ $result2 -eq 0 ]; then
        echo -e "${GREEN}✅ CRUD Operations: PASSED${NC}"
    else
        echo -e "${RED}❌ CRUD Operations: FAILED${NC}"
    fi
    
    if [ $result3 -eq 0 ]; then
        echo -e "${GREEN}✅ Redis Stress: PASSED${NC}"
    else
        echo -e "${RED}❌ Redis Stress: FAILED${NC}"
    fi
    
    echo -e "${YELLOW}📁 Results saved in: ${RESULTS_DIR}${NC}"
}

# Main menu loop
while true; do
    show_menu
    read -p "Enter your choice [0-6]: " choice
    
    case $choice in
        1)
            run_test "basic_performance" "rate_limiter_basic.js" "Tests core rate limiting with increasing load"
            ;;
        2)
            run_test "crud_operations" "api_crud_operations.js" "Tests all API endpoints with CRUD operations"
            ;;
        3)
            run_test "redis_stress" "redis_stress_test.js" "Tests Redis backend under high concurrent load"
            ;;
        4)
            run_test "rate_limiting_accuracy" "rate_limiting_accuracy.js" "Verifies rate limit enforcement accuracy"
            ;;
        5)
            run_all_sequential
            ;;
        6)
            echo -e "${YELLOW}⚠️  Parallel testing puts maximum stress on your system.${NC}"
            read -p "Are you sure you want to continue? (y/N): " confirm
            if [[ $confirm =~ ^[Yy]$ ]]; then
                run_all_parallel
            fi
            ;;
        0)
            echo -e "${GREEN}👋 Goodbye!${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}❌ Invalid choice. Please try again.${NC}"
            ;;
    esac
    
    echo ""
    read -p "Press Enter to continue..."
done