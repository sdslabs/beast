GO := go
pkgs  = $(shell $(GO) list ./... | grep -v vendor)


# Build beast
build:
	@./build/build.sh

# Check go formatting
check_format:
	@echo "[*] Checking for formatting errors using gofmt"
	@./build/check_gofmt.sh

# Add more tests later on for this
test: check_format

# Format code using gofmt
format:
	@echo "[*] Formatting code"
	@$(GO) fmt $(pkgs)


.PHONY: build format test check_format
