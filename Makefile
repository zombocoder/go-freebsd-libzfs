.PHONY: help build test test-destructive vet fmt fmt-check examples clean install deps docs check

GO ?= go
PACKAGES = ./...

help:
	@echo "FreeBSD Go ZFS Library"
	@echo "======================"
	@echo ""
	@echo "Available targets:"
	@echo "  build             - Build all packages and examples"
	@echo "  test              - Run tests that do not modify the system"
	@echo "  test-destructive  - Run all tests, including ones that create pools (root)"
	@echo "  vet               - Run go vet"
	@echo "  fmt               - Format code"
	@echo "  fmt-check         - Fail if any file is not gofmt-formatted"
	@echo "  examples          - Build example programs into ./bin"
	@echo "  clean             - Remove build artifacts"
	@echo "  install           - Download and tidy module dependencies"
	@echo "  deps              - Check system dependencies"
	@echo "  check             - fmt-check, vet and test"
	@echo "  docs              - Print package documentation"
	@echo ""

build:
	$(GO) build $(PACKAGES)

test:
	$(GO) test -count=1 $(PACKAGES)

# Creates and destroys throwaway file-backed pools under /var/tmp. Packages
# run one at a time because tests observe global pool state.
test-destructive:
	@if [ "$$(id -u)" != 0 ]; then echo "test-destructive must run as root (e.g. doas make test-destructive)"; exit 1; fi
	ZFS_TEST_DESTRUCTIVE=1 $(GO) test -count=1 -p 1 $(PACKAGES)

vet:
	$(GO) vet $(PACKAGES)

fmt:
	gofmt -s -w .

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "Not gofmt-formatted:"; echo "$$unformatted"; exit 1; fi

examples:
	@mkdir -p bin
	@for dir in examples/*/; do \
		name=$$(basename $$dir); \
		echo "Building $$name"; \
		$(GO) build -o bin/$$name ./$$dir || exit 1; \
	done

clean:
	rm -rf bin

install:
	$(GO) mod download
	$(GO) mod tidy

deps:
	@echo "Checking system dependencies..."
	@echo -n "Go version: "
	@$(GO) version || (echo "Error: Go not found" && exit 1)
	@echo -n "FreeBSD version: "
	@freebsd-version || (echo "Error: Not running on FreeBSD" && exit 1)
	@echo -n "ZFS kernel module: "
	@kldstat -q -m zfs && echo "loaded" || echo "not loaded"
	@echo -n "libzfs library: "
	@ldconfig -r | grep -q libzfs && echo "found" || echo "not found"
	@echo -n "OpenZFS headers: "
	@[ -f "/usr/src/sys/contrib/openzfs/include/libzfs.h" ] && echo "found" || echo "not found (see CONTRIBUTING.md)"

check: fmt-check vet test

docs:
	$(GO) doc -all ./zfs ./zpool ./version ./errors
