.PHONY: build test integration lint vet clean

BINARY := ai-spec
GOFLAGS := -trimpath

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINARY) .

test:
	go test ./...

integration:
	go test -tags=integration ./internal/integration/...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
