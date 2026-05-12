BINARY   := rren
PREFIX   ?= /usr/local
BINDIR   := $(PREFIX)/bin
GOFLAGS  ?=

.PHONY: all build install uninstall clean test lint

## all: build the binary (default target)
all: build

## build: compile the binary into the current directory
build:
	go build $(GOFLAGS) -o $(BINARY) .

## install: build and install the binary to $(BINDIR)
install: build
	install -d $(BINDIR)
	install -m 0755 $(BINARY) $(BINDIR)/$(BINARY)

## uninstall: remove the installed binary
uninstall:
	rm -f $(BINDIR)/$(BINARY)

## test: run all tests
test:
	go test ./...

## lint: run go vet
lint:
	go vet ./...

## help: print this help message
help:
	@sed -n 's/^## //p' $(MAKEFILE_LIST)
