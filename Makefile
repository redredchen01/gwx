APP_NAME := gwx
VERSION := 0.1.0
BUILD_DIR := ./build
MAIN := ./cmd/gwx

.PHONY: build install test vet clean help

## build: Build the binary
build:
	go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)

## install: Install to $GOPATH/bin
install:
	go install -ldflags "-s -w" $(MAIN)

## test: Run all tests with race detector
test:
	go test -race -count=1 ./...

## test-v: Run all tests verbose
test-v:
	go test -race -v ./...

## vet: Run go vet
vet:
	go vet ./...

## check: Run vet + test
check: vet test

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(APP_NAME)

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'

