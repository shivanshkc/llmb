SHELL=/usr/bin/env bash

# Project specific properties.
application_name        = llmb
application_binary_name = llmb

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

