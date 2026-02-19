# .juki/engine/Makefile

# Run dev server with Air (hot reload)
dev:
	go tool air

# Build binary
build:
	go build -o bin/engine cmd/main.go

# Tidy modules
tidy:
	go mod tidy

# Run tests
test:
	go test ./...

# Run linter (if installed)
lint:
	golangci-lint run
