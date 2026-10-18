.PHONY: help server worker start start-topic test build clean

help:
	@echo "Available commands:"
	@echo "  make server      - Start the local Temporal dev server (with Web UI at http://localhost:8233)"
	@echo "  make worker      - Run the Temporal worker"
	@echo "  make start       - Start an AI research workflow"
	@echo "  make build       - Build worker and starter binaries"
	@echo "  make clean       - Remove built binaries and output files"

server:
	temporal server start-dev

worker:
	go run worker/main.go

start:
	go run starter/main.go

start-topic:
	go run starter/main.go -topic "$(TOPIC)" -audience "$(AUDIENCE)"

build:
	go build -o bin/worker worker/main.go
	go build -o bin/starter starter/main.go

clean:
	rm -rf bin output/*.md
