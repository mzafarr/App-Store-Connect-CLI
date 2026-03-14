BINARY_NAME := gplay
GO := go
GOLANGCI_LINT_TIMEOUT ?= 5m
DIST_DIR ?= dist
VERSION ?= dev

.PHONY: build
build:
	$(GO) build -o $(BINARY_NAME) .

.PHONY: dist
dist:
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "-X main.version=$(VERSION)" -o $(DIST_DIR)/gplay_$(VERSION)_darwin_amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "-X main.version=$(VERSION)" -o $(DIST_DIR)/gplay_$(VERSION)_darwin_arm64 .
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "-X main.version=$(VERSION)" -o $(DIST_DIR)/gplay_$(VERSION)_linux_amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags "-X main.version=$(VERSION)" -o $(DIST_DIR)/gplay_$(VERSION)_linux_arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "-X main.version=$(VERSION)" -o $(DIST_DIR)/gplay_$(VERSION)_windows_amd64.exe .
	cd $(DIST_DIR) && \
		tar -czf gplay_$(VERSION)_darwin_amd64.tar.gz gplay_$(VERSION)_darwin_amd64 && \
		tar -czf gplay_$(VERSION)_darwin_arm64.tar.gz gplay_$(VERSION)_darwin_arm64 && \
		tar -czf gplay_$(VERSION)_linux_amd64.tar.gz gplay_$(VERSION)_linux_amd64 && \
		tar -czf gplay_$(VERSION)_linux_arm64.tar.gz gplay_$(VERSION)_linux_arm64 && \
		zip -q gplay_$(VERSION)_windows_amd64.zip gplay_$(VERSION)_windows_amd64.exe && \
		rm -f gplay_$(VERSION)_darwin_amd64 gplay_$(VERSION)_darwin_arm64 gplay_$(VERSION)_linux_amd64 gplay_$(VERSION)_linux_arm64 gplay_$(VERSION)_windows_amd64.exe && \
		shasum -a 256 *.tar.gz *.zip > checksums.txt

.PHONY: test
test:
	$(GO) test ./...

.PHONY: lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=$(GOLANGCI_LINT_TIMEOUT) ./...; \
	else \
		echo "golangci-lint not found; falling back to 'go vet ./...'"; \
		$(GO) vet ./...; \
	fi

.PHONY: format
format:
	$(GO) fmt ./...
	gofmt -w .

.PHONY: format-check
format-check:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Formatting issues detected:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
