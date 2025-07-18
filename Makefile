# MimirInsights Makefile

# Variables
APP_NAME := mimir-insights
VERSION := $(shell git describe --tags --always --dirty)
TIMESTAMP := $(shell date +%Y%m%d-%H%M%S)
REGISTRY := ghcr.io/akshaydubey29
NAMESPACE := mimir-insights

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Docker variables
DOCKER_BUILD := docker build
DOCKER_PUSH := docker push
DOCKER_TAG := docker tag

# Helm variables
HELM := helm
HELM_INSTALL := $(HELM) install
HELM_UPGRADE := $(HELM) upgrade
HELM_UNINSTALL := $(HELM) uninstall

.PHONY: all build test clean deps lint docker-build docker-push docker-tag helm-install helm-upgrade helm-uninstall help

# Default target
all: clean deps test build

# Build the application
build:
	@echo Building MimirInsights..."
	$(GOBUILD) -o bin/server ./cmd/server
	@echo "Build complete!"

# Run tests
test:
	@echo "Running tests...
	$(GOTEST) -v ./...
	@echo "Tests complete!"

# Clean build artifacts
clean:
	@echo Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	@echo "Clean complete!"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies complete!"

# Run linting
lint:
	@echo "Running linter..."
	golangci-lint run
	@echoLinting complete!"

# Build Docker images with timestamp tags
docker-build:
	@echo "Building Docker images with timestamp $(TIMESTAMP)..."
	$(DOCKER_BUILD) -f Dockerfile.backend -t $(REGISTRY)/$(APP_NAME)-backend:$(TIMESTAMP) .
	$(DOCKER_BUILD) -f Dockerfile.frontend -t $(REGISTRY)/$(APP_NAME)-ui:$(TIMESTAMP) .
	@echo "Docker build complete!"

# Tag Docker images with latest
docker-tag:
	@echo "Tagging Docker images as latest..."
	$(DOCKER_TAG) $(REGISTRY)/$(APP_NAME)-backend:$(TIMESTAMP) $(REGISTRY)/$(APP_NAME)-backend:latest
	$(DOCKER_TAG) $(REGISTRY)/$(APP_NAME)-ui:$(TIMESTAMP) $(REGISTRY)/$(APP_NAME)-ui:latest
	@echo "Docker tagging complete!"

# Push Docker images
docker-push:
	@echo "Pushing Docker images..."
	$(DOCKER_PUSH) $(REGISTRY)/$(APP_NAME)-backend:$(TIMESTAMP)
	$(DOCKER_PUSH) $(REGISTRY)/$(APP_NAME)-backend:latest
	$(DOCKER_PUSH) $(REGISTRY)/$(APP_NAME)-ui:$(TIMESTAMP)
	$(DOCKER_PUSH) $(REGISTRY)/$(APP_NAME)-ui:latest
	@echo "Docker push complete!"

# Build and push Docker images
docker: docker-build docker-tag docker-push

# Multi-architecture build targets
multi-arch-build:
	@echo "Building multi-architecture Docker images..."
	./build-multi-arch.sh

multi-arch-build-local:
	@echo "Building multi-architecture Docker images for local testing..."
	./build-multi-arch-local.sh

multi-arch-test:
	@echo "Testing multi-architecture support..."
	./test-multi-arch.sh

multi-arch-deploy:
	@echo "Deploying multi-architecture images..."
	./deploy-multi-arch.sh

# Install Helm chart
helm-install:
	@echo "Installing Helm chart..."
	$(HELM_INSTALL) $(APP_NAME) ./deployments/helm-chart \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set backend.image.tag=$(TIMESTAMP) \
		--set frontend.image.tag=$(TIMESTAMP)
	@echo "Helm install complete!"

# Upgrade Helm chart
helm-upgrade:
	@echo "Upgrading Helm chart..."
	$(HELM_UPGRADE) $(APP_NAME) ./deployments/helm-chart \
		--namespace $(NAMESPACE) \
		--set backend.image.tag=$(TIMESTAMP) \
		--set frontend.image.tag=$(TIMESTAMP)
	@echo "Helm upgrade complete!"

# Uninstall Helm chart
helm-uninstall:
	@echo "Uninstalling Helm chart..."
	$(HELM_UNINSTALL) $(APP_NAME) --namespace $(NAMESPACE)
	@echo "Helm uninstall complete!"

# Deploy to Kubernetes
deploy: docker helm-upgrade

# Run locally
run:
	@echoRunning MimirInsights locally..."
	$(GOBUILD) -o bin/server ./cmd/server
	./bin/server

# Run with hot reload (requires air)
dev:
	@echo "Running in development mode...
	air

# Generate documentation
docs:
	@echo "Generating documentation..."
	swag init -g cmd/server/main.go
	@echo "Documentation generated!

# Security scan
security-scan:
	@echoRunning security scan..."
	trivy image $(REGISTRY)/$(APP_NAME)-backend:$(TIMESTAMP)
	trivy image $(REGISTRY)/$(APP_NAME)-ui:$(TIMESTAMP)
	@echo "Security scan complete!"

# Format code
fmt:
	@echo "Formatting code...
	gofmt -s -w .
	@echo "Code formatting complete!"

# Check code quality
check: fmt lint test

# Setup development environment
setup:
	@echo "Setting up development environment..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Development environment setup complete!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  lint           - Run linting"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-tag     - Tag Docker images"
	@echo "  docker-push    - Push Docker images"
	@echo "  docker         - Build and push Docker images"
	@echo "  helm-install   - Install Helm chart"
	@echo "  helm-upgrade   - Upgrade Helm chart"
	@echo "  helm-uninstall - Uninstall Helm chart"
	@echo "  deploy         - Deploy to Kubernetes"
	@echo "  run            - Run locally"
	@echo "  dev            - Run with hot reload"
	@echo "  docs           - Generate documentation"
	@echo "  security-scan  - Run security scan"
	@echo "  fmt            - Format code"
	@echo "  check          - Format, lint, and test"
	@echo "  setup          - Setup development environment"
	@echo "  help           - Show this help"
	@echo "  multi-arch-build      - Build multi-architecture images"
	@echo "  multi-arch-build-local - Build multi-architecture images for local testing"
	@echo "  multi-arch-test       - Test multi-architecture support"
	@echo "  multi-arch-deploy     - Deploy multi-architecture images" 