#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🧪 Running test coverage analysis..."

# Create coverage directory
mkdir -p coverage

# Run tests with coverage
echo "📊 Running unit tests with coverage..."
go test -v -race -coverprofile=coverage/coverage.out -covermode=atomic ./...

# Generate HTML coverage report
echo "📄 Generating HTML coverage report..."
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Generate coverage summary
echo -e "\n${GREEN}📈 Coverage Summary:${NC}"
go tool cover -func=coverage/coverage.out | tail -n 1

# Run coverage by package
echo -e "\n${YELLOW}📦 Coverage by Package:${NC}"
go tool cover -func=coverage/coverage.out | grep -E "^github.com/vaultenv" | sort -k3 -nr

# Check coverage threshold
COVERAGE_THRESHOLD=80
TOTAL_COVERAGE=$(go tool cover -func=coverage/coverage.out | tail -n 1 | awk '{print $3}' | sed 's/%//')
TOTAL_COVERAGE_INT=${TOTAL_COVERAGE%.*}

echo -e "\n${YELLOW}🎯 Coverage Threshold Check:${NC}"
echo "Threshold: ${COVERAGE_THRESHOLD}%"
echo "Actual: ${TOTAL_COVERAGE}%"

if [ $TOTAL_COVERAGE_INT -ge $COVERAGE_THRESHOLD ]; then
    echo -e "${GREEN}✅ Coverage threshold met!${NC}"
else
    echo -e "${RED}❌ Coverage below threshold!${NC}"
    exit 1
fi

# Run integration tests if requested
if [ "$1" == "--integration" ]; then
    echo -e "\n${YELLOW}🔗 Running integration tests...${NC}"
    go test -v -tags=integration -timeout=10m ./...
fi

# Run benchmarks if requested
if [ "$1" == "--bench" ] || [ "$2" == "--bench" ]; then
    echo -e "\n${YELLOW}⚡ Running benchmarks...${NC}"
    go test -bench=. -benchmem -run=^$ ./... | tee coverage/benchmark.txt
fi

echo -e "\n${GREEN}✨ Coverage analysis complete!${NC}"
echo "📂 HTML report: coverage/coverage.html"