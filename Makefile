.PHONY: check-spec-paths build test integration lint vet clean coverage coverage-packages fuzz bench budget check-skills-sync check-hooks-sync check-scripts-sync test-hooks test-validators test-sdd-evals smoke-adapters test-portable-skills sync-acp-sdk-version test-acp-live test-hooks-live mocks check-mocks test-check-mocks

BINARY := ai-spec
GOFLAGS := -trimpath
MOCKERY_VERSION := v3.7.4

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINARY) .

# mocks: (re)gera os mocks declarados em mockery.yml via mockery.
# Pos-processa para usar 'any' no lugar de 'interface{}' (Regra 7.1).
mocks:
	go run github.com/vektra/mockery/v3@$(MOCKERY_VERSION) --config mockery.yml
	bash scripts/normalize-mocks.sh

# check-mocks: falha se os mocks estiverem desatualizados em relacao as interfaces.
check-mocks:
	bash scripts/check-mocks.sh

test-check-mocks:
	bash scripts/check-mocks_test.sh

test:
	go test ./...

integration:
	go test -tags=integration ./internal/integration/... ./internal/skills/... ./tests/integration/...

lint:
	@echo "Running linter..."
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2
	GOGC=20 golangci-lint run --config .golangci.yml --timeout 10m --verbose

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

coverage:
	go test -coverprofile=coverage.out ./...
	grep -v '/mocks/' coverage.out > coverage_filtered.out
	go tool cover -func=coverage_filtered.out

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
	go test -fuzz=FuzzTranslator -fuzztime=30s ./internal/approval/
	go test -fuzz=FuzzFingerprint -fuzztime=30s ./internal/approval/

bench:
	go test -bench=. -benchmem ./internal/metrics/ ./internal/skills/ ./internal/parity/

budget:
	go test -tags=integration -run TestTokenBudget ./internal/integration/...

check-skills-sync:
	bash scripts/check-skills-sync.sh

check-hooks-sync:
	bash scripts/check-hooks-sync.sh

check-scripts-sync:
	bash scripts/check-scripts-sync.sh

test-hooks:
	bash scripts/test-hooks.sh

# check-spec-paths: falha se artefato de contrato sob gestao SDD citar caminho
# que nao existe. Provado nos dois sentidos por tests/scripts/.
check-spec-paths:
	bash scripts/check-spec-paths.sh
	bash tests/scripts/check-spec-paths_test.sh

test-validators:
	bash scripts/test-validators.sh
	bash tests/scripts/validate-task-evidence_test.sh .agents/scripts/validate-task-evidence.sh
	bash tests/scripts/validate-bugfix-evidence_test.sh .agents/scripts/validate-bugfix-evidence.sh

test-sdd-evals:
	bash scripts/test-sdd-evals.sh

smoke-adapters:
	go test -count=1 ./internal/adapters/... ./internal/taskloop/... -run 'TestGenerate_executeTaskYAMLContract_allTools|TestE2EAgent_PromptContainsAgentBlocks'

test-portable-skills:
	bash scripts/test-portable-skills.sh

# sync-acp-sdk-version: mantém ClaudeSDKVersion em internal/runtime/specs/claude.go
# sincronizada com a versão de github.com/coder/acp-go-sdk declarada em go.mod.
# Rodar localmente após atualizar go.mod. Não incluído em CI automaticamente (ADR-009).
sync-acp-sdk-version:
	bash scripts/sync-acp-sdk-version.sh

# test-acp-live: executa os testes live do runtime ACP.
# Requer claude-agent-acp ou npx disponíveis no PATH (ver tests/integration/acp_live/README.md).
# Não incluído em make test (build tag acp_live protege compilação). Rodado pelo CI nightly.
test-acp-live:
	go test -tags=acp_live -v ./tests/integration/acp_live

# test-hooks-live: executa a camada 3 de prova de enforcement dos hooks nativos.
# Requer os binarios/CLIs de cada agente disponiveis no PATH (ver tests/integration/hooks_live/README.md).
# Nao incluido em make test (build tag hooks_live protege compilacao). Rodado pelo CI nightly; nao e gate de merge.
test-hooks-live:
	go test -tags=hooks_live -v ./tests/integration/hooks_live
