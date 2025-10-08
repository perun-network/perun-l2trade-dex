.PHONY: build run clean test deps

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build settings
BINARY_NAME=perun-dex-server
BINARY_UNIX=$(BINARY_NAME)_unix
CMD_PATH=./cmd/server

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v $(CMD_PATH)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(CMD_PATH)

run:
	$(GOBUILD) -o $(BINARY_NAME) -v $(CMD_PATH)
	./$(BINARY_NAME)

run-dev:
	$(GOCMD) run $(CMD_PATH)/main.go

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

test:
	$(GOTEST) -v ./...

deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Docker commands (optional)
docker-build:
	docker build -t perun-dex-websocket .

docker-run:
	docker run -p 8080:8080 perun-dex-websocket

# Development helpers
install-deps:
	$(GOGET) github.com/gorilla/websocket
	$(GOGET) github.com/google/uuid
	$(GOGET) github.com/sirupsen/logrus

format:
	$(GOCMD) fmt ./...

vet:
	$(GOCMD) vet ./...

# Quick start for demo
demo: deps build
	@echo "Starting Perun DEX WebSocket Demo..."
	@echo "Open your browser to http://localhost:8080"
	./$(BINARY_NAME)