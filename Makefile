.PHONY: help build run wire test lint migrate clean

# Default target
.DEFAULT_GOAL := help

APP_NAME := ai-adp
BUILD_DIR := bin
CONFIG    := configs/app.yaml

## help: Show available targets
help:
	@echo "AI Development Platform — available targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  /'

## build: Build the binary to bin/ai-adp
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "Binary: $(BUILD_DIR)/$(APP_NAME)"

## run: Build and start the server
run: build
	./$(BUILD_DIR)/$(APP_NAME) serve --config $(CONFIG)

## wire: Regenerate Google Wire DI code
wire:
	@echo "Generating Wire DI code..."
	cd internal/di && wire
	@echo "Done."

## test: Run all tests
test:
	go test ./... -count=1 -race -timeout 120s

## test-cover: Run tests with coverage report
test-cover:
	go test ./... -count=1 -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## migrate: Run database migrations
migrate: build
	./$(BUILD_DIR)/$(APP_NAME) migrate --config $(CONFIG)

## mockgen: Regenerate all mocks (add targets as needed)
mockgen:
	@echo "Regenerating mocks..."
	mockgen -source=internal/domain/tenant/repository/tenant_repo.go \
		-destination=internal/domain/tenant/repository/mock/mock_tenant_repo.go \
		-package=mockrepo
	mockgen -source=internal/application/tenant/service/tenant_app_service.go \
		-destination=internal/application/tenant/service/mock/mock_tenant_svc.go \
		-package=mocksvc
	@echo "Mocks regenerated."

## clean: Remove build artifacts
clean:
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "Cleaned."

## vet: Run go vet
vet:
	go vet ./...

## tidy: Tidy go modules
tidy:
	go mod tidy
