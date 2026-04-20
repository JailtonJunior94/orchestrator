.PHONY: build test integration lint vet clean coverage coverage-packages fuzz bench

BINARY := ai-spec
GOFLAGS := -trimpath

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINARY) .

test:
	go test ./...

integration:
	go test -tags=integration ./internal/integration/... ./internal/skills/...

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

coverage-packages:
	bash scripts/check-package-coverage.sh 70

fuzz:
	go test -fuzz=FuzzParseFrontmatter -fuzztime=30s ./internal/skills/
	go test -fuzz=FuzzValidateFrontmatter -fuzztime=30s ./internal/skills/
	go test -fuzz=FuzzParseTaskFile -fuzztime=30s ./internal/taskloop/
	go test -fuzz=FuzzReadTaskFileStatus -fuzztime=30s ./internal/taskloop/
	go test -fuzz=FuzzValidateBugReport -fuzztime=30s ./internal/bugschema/

bench:
	go test -bench=. -benchmem ./internal/metrics/ ./internal/skills/ ./internal/parity/
