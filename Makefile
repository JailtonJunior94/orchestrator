.PHONY: build test integration lint vet clean coverage coverage-packages fuzz bench budget check-skills-sync check-hooks-sync test-hooks

BINARY := ai-spec
GOFLAGS := -trimpath

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINARY) .

test:
	go test ./...

integration:
	go test -tags=integration ./internal/integration/... ./internal/skills/...

lint:
	@echo "Running linter..."
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.2.1
	GOGC=20 golangci-lint run --config .golangci.yml --timeout 10m --verbose

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
	go test -fuzz=FuzzParseConfig -fuzztime=30s ./internal/config/
	go test -fuzz=FuzzParseManifest -fuzztime=30s ./internal/manifest/
	go test -fuzz=FuzzDetectLanguages -fuzztime=30s ./internal/detect/
	go test -fuzz=FuzzDetectToolchain -fuzztime=30s ./internal/detect/

bench:
	go test -bench=. -benchmem ./internal/metrics/ ./internal/skills/ ./internal/parity/

budget:
	go test -tags=integration -run TestTokenBudget ./internal/integration/...

check-skills-sync:
	bash scripts/check-skills-sync.sh

check-hooks-sync:
	bash scripts/check-hooks-sync.sh

test-hooks:
	bash scripts/test-hooks.sh
