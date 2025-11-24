VERSION ?= $(shell git describe --tags --always)
GOBIN ?= $(shell go env GOPATH)/bin

# Integration test configuration
BINARY ?= bin/frameo-miniatures
TEST_INPUT := example
TEST_OUTPUT_BASE := test-output

# Test targets
.PHONY: run build install \
		integration-test test-version test-basic-webp test-jpeg-output test-custom-resolution \
		test-dry-run test-skip-existing test-prune test-skip-existing-prune integration-test-clean test-unit

run:
	go run \
		-ldflags="-X main.BuildVersion=$(VERSION)" \
		./cmd

build: bin/frameo-miniatures

bin/frameo-miniatures:
	go build \
		-ldflags="-X main.BuildVersion=$(VERSION)" \
		-o bin/frameo-miniatures

install: bin/frameo-miniatures
	@mkdir -p $(GOBIN)
	@cp bin/frameo-miniatures $(GOBIN)/frameo-miniatures
	@echo "Installed frameo-miniatures to $(GOBIN)/frameo-miniatures"

$(GOBIN)/goimports:
	@go install golang.org/x/tools/cmd/goimports@v0.35.0

$(GOBIN)/gocyclo:
	@go install github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0

$(GOBIN)/golangci-lint:
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.0

$(GOBIN)/gocritic:
	@go install github.com/go-critic/go-critic/cmd/gocritic@v0.13.0

install-linters: $(GOBIN)/goimports $(GOBIN)/gocyclo $(GOBIN)/golangci-lint $(GOBIN)/gocritic
	@echo "Linters installed successfully."

lint: install-linters
	@pre-commit run -a

clean:
	@rm -rfv bin
	@rm -fv frameo-miniatures
	@rm -rfv test-*

test: test-unit integration-test
	@echo ""
	@echo "✓ All tests passed!"

test-unit:
	go test -v ./...

integration-test: bin/frameo-miniatures test-version test-basic-webp test-jpeg-output \
                  test-custom-resolution test-dry-run test-skip-existing test-prune \
                  test-skip-existing-prune integration-test-clean
	@echo ""
	@echo "✓ All integration tests passed!"

test-version:
	@echo "[TEST] Version check"
	@$(BINARY) --version > /dev/null 2>&1 && echo "  ✓ PASSED" || (echo "  ✗ FAILED" && exit 1)

test-basic-webp:
	@echo "[TEST] Basic WebP processing"
	@rm -rf $(TEST_OUTPUT_BASE)/basic-webp
	@$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/basic-webp > /dev/null 2>&1
	@test -f $(TEST_OUTPUT_BASE)/basic-webp/*.webp && echo "  ✓ PASSED" || (echo "  ✗ FAILED: No WebP files created" && exit 1)

test-jpeg-output:
	@echo "[TEST] JPEG output format"
	@rm -rf $(TEST_OUTPUT_BASE)/jpeg-output
	@$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/jpeg-output -f jpg -q 85 > /dev/null 2>&1
	@test -f $(TEST_OUTPUT_BASE)/jpeg-output/*.jpg && echo "  ✓ PASSED" || (echo "  ✗ FAILED: No JPEG files created" && exit 1)

test-custom-resolution:
	@echo "[TEST] Custom resolution"
	@rm -rf $(TEST_OUTPUT_BASE)/custom-resolution
	@$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/custom-resolution -r 1920x1080 > /dev/null 2>&1
	@test -f $(TEST_OUTPUT_BASE)/custom-resolution/*.webp && echo "  ✓ PASSED" || (echo "  ✗ FAILED: No files created" && exit 1)

test-dry-run:
	@echo "[TEST] Dry-run mode"
	@rm -rf $(TEST_OUTPUT_BASE)/dry-run
	@$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/dry-run --dry-run > /dev/null 2>&1
	@if [ -d $(TEST_OUTPUT_BASE)/dry-run ] && [ -n "$$(find $(TEST_OUTPUT_BASE)/dry-run -type f 2>/dev/null)" ]; then \
		echo "  ✗ FAILED: Files created in dry-run mode"; exit 1; \
	else \
		echo "  ✓ PASSED"; \
	fi

test-skip-existing:
	@echo "[TEST] Skip existing files"
	@rm -rf $(TEST_OUTPUT_BASE)/skip-existing
	@$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/skip-existing > /dev/null 2>&1
	@FIRST_FILE=$$(find $(TEST_OUTPUT_BASE)/skip-existing -type f -name "*.webp" | head -n 1); \
	if [ -z "$$FIRST_FILE" ]; then echo "  ✗ FAILED: No files created"; exit 1; fi; \
	MTIME_BEFORE=$$(stat -c %Y "$$FIRST_FILE" 2>/dev/null || stat -f %m "$$FIRST_FILE" 2>/dev/null); \
	sleep 2; \
	$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/skip-existing --skip-existing > /dev/null 2>&1; \
	MTIME_AFTER=$$(stat -c %Y "$$FIRST_FILE" 2>/dev/null || stat -f %m "$$FIRST_FILE" 2>/dev/null); \
	if [ "$$MTIME_BEFORE" -eq "$$MTIME_AFTER" ]; then \
		echo "  ✓ PASSED"; \
	else \
		echo "  ✗ FAILED: File was modified"; exit 1; \
	fi

test-prune:
	@echo "[TEST] Prune orphaned files"
	@rm -rf $(TEST_OUTPUT_BASE)/prune
	@$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/prune > /dev/null 2>&1
	@touch $(TEST_OUTPUT_BASE)/prune/orphaned-file.webp
	@$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/prune --prune > /dev/null 2>&1
	@if [ -f $(TEST_OUTPUT_BASE)/prune/orphaned-file.webp ]; then \
		echo "  ✗ FAILED: Orphaned file not removed"; exit 1; \
	else \
		echo "  ✓ PASSED"; \
	fi

test-skip-existing-prune:
	@echo "[TEST] Skip-existing with prune"
	@rm -rf $(TEST_OUTPUT_BASE)/skip-existing-prune
	@$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/skip-existing-prune > /dev/null 2>&1
	@touch $(TEST_OUTPUT_BASE)/skip-existing-prune/orphaned-file.webp
	@FIRST_FILE=$$(find $(TEST_OUTPUT_BASE)/skip-existing-prune -type f -name "*.webp" ! -name "orphaned-file.webp" | head -n 1); \
	MTIME_BEFORE=$$(stat -c %Y "$$FIRST_FILE" 2>/dev/null || stat -f %m "$$FIRST_FILE" 2>/dev/null); \
	sleep 2; \
	$(BINARY) -i $(TEST_INPUT) -o $(TEST_OUTPUT_BASE)/skip-existing-prune --skip-existing --prune > /dev/null 2>&1; \
	MTIME_AFTER=$$(stat -c %Y "$$FIRST_FILE" 2>/dev/null || stat -f %m "$$FIRST_FILE" 2>/dev/null); \
	if [ ! -f $(TEST_OUTPUT_BASE)/skip-existing-prune/orphaned-file.webp ] && [ "$$MTIME_BEFORE" -eq "$$MTIME_AFTER" ]; then \
		echo "  ✓ PASSED"; \
	else \
		echo "  ✗ FAILED"; exit 1; \
	fi

integration-test-clean:
	@echo ""
	@echo "Cleaning up test directories..."
	@$(MAKE) clean
