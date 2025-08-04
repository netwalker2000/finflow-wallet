# Makefile

.PHONY: lint test build all clean

# Define variables
APP_NAME := finflow-wallet
BUILD_DIR := bin
MAIN_GO := ./cmd/api/main.go
TEST_COVERAGE_DIR := coverage

# Default target
all: lint test build

# Run linters
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=5m --issues-exit-code=1 || (echo "Linting failed!" && exit 1)
	@echo "Linting successful."

# Run unit tests with coverage
test:
	@echo "Running Go tests..."
	@mkdir -p $(TEST_COVERAGE_DIR)
	@go test -v -race -coverprofile=$(TEST_COVERAGE_DIR)/coverage.out ./... || (echo "Tests failed!" && exit 1)
	@go tool cover -html=$(TEST_COVERAGE_DIR)/coverage.out -o $(TEST_COVERAGE_DIR)/coverage.html
	@echo "Tests successful. Coverage report generated at $(TEST_COVERAGE_DIR)/coverage.html"

# Build the application binary
build:
	@echo "Building application..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_GO) || (echo "Build failed!" && exit 1)
	@echo "Build successful: $(BUILD_DIR)/$(APP_NAME)"

# Clean build artifacts and test coverage reports
clean:
	@echo "Cleaning build artifacts and test coverage reports..."
	@rm -rf $(BUILD_DIR) $(TEST_COVERAGE_DIR)
	@echo "Clean complete."

# Help message
help:
	@echo "Usage:"
	@echo "  make all    - Run lint, tests, and build the application."
	@echo "  make lint   - Run static code analysis (golangci-lint)."
	@echo "  make test   - Run unit tests with race detector and generate coverage report."
	@echo "  make build  - Build the application binary."
	@echo "  make clean  - Remove build artifacts and test coverage reports."
	@echo "  make help   - Display this help message."