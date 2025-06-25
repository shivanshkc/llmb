SHELL=/usr/bin/env bash

# Project specific properties.
application_name        = llmb
application_binary_name = llmb

url    = "http://localhost:8080"
prompt = "Generate a comma-separated list of all prime numbers between 30 and 60, nothing else."

# Builds the project.
build:
	@echo "+$@"
	@go build -o bin/$(application_binary_name) cmd/$(application_name)/main.go

# Tests the whole project.
test:
	@echo "+$@"
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Runs the "go mod tidy" command.
tidy:
	@echo "+$@"
	@go mod tidy

# Runs golang-ci-lint over the project.
lint:
	@echo "+$@"
	@golangci-lint run ./...

chat: build
	@echo "+$@"
	@bin/llmb chat -u $(url) -m llama3.1

bench: build
	@echo "+$@"
	@bin/llmb bench -u $(url) -m llama3.1 -p $(prompt)