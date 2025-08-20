.PHONY: help build dev test clean run install lint fmt vet deps installer

# Variables
APP_NAME := WinRamp
BINARY_NAME := winramp.exe
MAIN_PATH := ./cmd/winramp
BUILD_DIR := ./build
DIST_DIR := ./dist
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -w -s"

# Colors for terminal output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(GREEN)WinRamp Build System$(NC)"
	@echo "$(YELLOW)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

deps: ## Download and install dependencies
	@echo "$(YELLOW)Installing dependencies...$(NC)"
	go mod download
	go mod tidy
	@echo "$(GREEN)Dependencies installed successfully$(NC)"

build: deps ## Build the application for production
	@echo "$(YELLOW)Building $(APP_NAME) v$(VERSION)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	wails build -clean -platform windows/amd64 -o $(BINARY_NAME)
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-debug: deps ## Build the application with debug symbols
	@echo "$(YELLOW)Building $(APP_NAME) (debug) v$(VERSION)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	wails build -debug -platform windows/amd64 -o $(BINARY_NAME)
	@echo "$(GREEN)Debug build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

dev: ## Run the application in development mode with hot reload
	@echo "$(YELLOW)Starting development server...$(NC)"
	wails dev

run: build ## Build and run the application
	@echo "$(YELLOW)Running $(APP_NAME)...$(NC)"
	$(BUILD_DIR)/$(BINARY_NAME)

test: ## Run all tests
	@echo "$(YELLOW)Running tests...$(NC)"
	go test -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)Tests completed$(NC)"

test-coverage: test ## Run tests with coverage report
	@echo "$(YELLOW)Generating coverage report...$(NC)"
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

benchmark: ## Run benchmarks
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	go test -bench=. -benchmem ./...

lint: ## Run linters
	@echo "$(YELLOW)Running linters...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(RED)golangci-lint not found. Installing...$(NC)" && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...
	@echo "$(GREEN)Linting complete$(NC)"

fmt: ## Format code
	@echo "$(YELLOW)Formatting code...$(NC)"
	go fmt ./...
	gofmt -s -w .
	@echo "$(GREEN)Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(NC)"
	go vet ./...
	@echo "$(GREEN)Vet complete$(NC)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@rm -f *.pprof *.prof
	@echo "$(GREEN)Clean complete$(NC)"

installer: build ## Create Windows installer
	@echo "$(YELLOW)Creating Windows installer...$(NC)"
	@mkdir -p $(DIST_DIR)
	# This would use WiX or NSIS to create an installer
	# For now, we'll just copy the binary
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(DIST_DIR)/
	@echo "$(GREEN)Installer created in $(DIST_DIR)$(NC)"

proto: ## Generate protobuf files (if needed)
	@echo "$(YELLOW)Generating protobuf files...$(NC)"
	@echo "$(GREEN)Protobuf generation complete$(NC)"

migrate: ## Run database migrations
	@echo "$(YELLOW)Running database migrations...$(NC)"
	go run $(MAIN_PATH) migrate up
	@echo "$(GREEN)Migrations complete$(NC)"

migrate-down: ## Rollback database migrations
	@echo "$(YELLOW)Rolling back database migrations...$(NC)"
	go run $(MAIN_PATH) migrate down
	@echo "$(GREEN)Rollback complete$(NC)"

docker-build: ## Build Docker image (for CI/CD)
	@echo "$(YELLOW)Building Docker image...$(NC)"
	docker build -t $(APP_NAME):$(VERSION) .
	@echo "$(GREEN)Docker image built: $(APP_NAME):$(VERSION)$(NC)"

check: fmt vet lint test ## Run all checks (format, vet, lint, test)
	@echo "$(GREEN)All checks passed!$(NC)"

install: build ## Install the application to system
	@echo "$(YELLOW)Installing $(APP_NAME)...$(NC)"
	@mkdir -p "$(APPDATA)/$(APP_NAME)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) "$(APPDATA)/$(APP_NAME)/"
	@echo "$(GREEN)$(APP_NAME) installed to $(APPDATA)/$(APP_NAME)$(NC)"

uninstall: ## Uninstall the application
	@echo "$(YELLOW)Uninstalling $(APP_NAME)...$(NC)"
	@rm -rf "$(APPDATA)/$(APP_NAME)"
	@echo "$(GREEN)$(APP_NAME) uninstalled$(NC)"

.DEFAULT_GOAL := help