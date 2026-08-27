# Basic Makefile to install deps, build, and run the application.
# Usage:
#   make deps    # run `go mod tidy`
#   make build   # build binary into bin/
#   make run     # build and run the binary
#   make clean   # remove built artifacts

BINARY := bin/url-shortner
PKG := .

.PHONY: all deps build run clean

all: build

deps:
	go mod tidy

build: deps
	@mkdir -p bin
	go build -o $(BINARY) $(PKG)
	@echo "Built $(BINARY)"

run: build
	@echo "Running $(BINARY)"
	./$(BINARY)

clean:
	-rm -rf bin
	@echo "Cleaned"