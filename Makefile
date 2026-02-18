BINARY  := chatterbox
PKG     := github.com/devaloi/chatterbox
GOFLAGS := -race

.PHONY: build test lint run clean cover vet

build:
	go build -o bin/$(BINARY) $(PKG)/cmd/server

test:
	go test $(GOFLAGS) ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin/ *.db coverage.out coverage.html

cover:
	go test $(GOFLAGS) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
