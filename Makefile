.PHONY: help build test clean examples install deps fmt check docs

help:
	@echo "FreeBSD Go ZFS Library"
	@echo "======================"
	@echo ""
	@echo "Available targets:"
	@echo "  build             - Build all binaries"
	@echo "  test              - Run unit tests"
	@echo "  examples          - Build all example programs"
	@echo "  clean             - Clean build artifacts"
	@echo "  install           - Install as Go module"
	@echo "  deps              - Check system dependencies"
	@echo "  fmt               - Format code"
	@echo "  check             - Run all checks (fmt, test)"
	@echo "  docs              - Generate documentation"
	@echo ""

build: examples

test:
	@echo "Running unit tests for core library..."
	go test ./internal/driver ./zfs ./zpool ./version ./errors


examples:
	@echo "Building example programs..."
	@for dir in examples/*/; do \
		if [ -f "$$dir/main.go" ]; then \
			echo "Building $$dir..."; \
			(cd "$$dir" && go build -o "$$(basename $$dir)" main.go) || true; \
		fi \
	done

clean:
	@echo "Cleaning build artifacts..."
	find ./** -type f -executable -delete 2>/dev/null || true

install:
	@echo "Installing module dependencies..."
	go mod download
	go mod tidy

deps:
	@echo "Checking system dependencies..."
	@echo -n "Go version: "
	@go version || (echo "Error: Go not found" && exit 1)
	@echo -n "FreeBSD version: "
	@freebsd-version || (echo "Error: Not running on FreeBSD" && exit 1)
	@echo -n "ZFS kernel module: "
	@kldstat | grep zfs >/dev/null && echo "loaded" || echo "not loaded"
	@echo -n "libzfs library: "
	@ldconfig -r | grep libzfs >/dev/null && echo "found" || echo "not found"
	@echo -n "OpenZFS headers: "
	@[ -d "/usr/src/sys/contrib/openzfs/include" ] && echo "found" || echo "not found"

fmt:
	@echo "Formatting core library code..."
	go fmt ./internal/... ./zfs ./zpool ./version ./errors
	gofmt -s -w ./internal/ ./zfs/ ./zpool/ ./version/ ./errors/

check: fmt test
	@echo "All checks passed!"

docs:
	@echo "Generating documentation..."
	go doc -all ./...
