.PHONY: all build test vet cover smoke clean

BINARY := kt

all: vet test build

build:
	go build -o $(BINARY) ./cmd/kt

test:
	go test ./... -count=1

vet:
	go vet ./...

cover:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -func=coverage.out
	@rm -f coverage.out

smoke: build
	@echo "=== Smoke tests ==="
	@export TMPDIR=$$(mktemp -d) && \
	export XDG_DATA_HOME="$$TMPDIR/data" && \
	export XDG_CONFIG_HOME="$$TMPDIR/config" && \
	echo "  version..." && \
	./$(BINARY) --version && \
	echo "  add..." && \
	./$(BINARY) add "smoke test task" && \
	echo "  list..." && \
	./$(BINARY) list && \
	echo "  done (first task)..." && \
	TASK_ID=$$(./$(BINARY) list 2>/dev/null | grep -E '^\s+[0-9A-Z]{8}' | head -1 | awk '{print $$1}') && \
	./$(BINARY) done "$$TASK_ID" && \
	echo "  daemon status..." && \
	./$(BINARY) daemon status && \
	rm -rf "$$TMPDIR" && \
	echo "=== All smoke tests passed ==="

clean:
	rm -f $(BINARY) coverage.out
