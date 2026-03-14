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
	@echo "  version..." && \
	./$(BINARY) --version && \
	echo "  help..." && \
	./$(BINARY) --help && \
	echo "=== All smoke tests passed ==="

clean:
	rm -f $(BINARY) coverage.out
