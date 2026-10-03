.PHONY: help build fmt fmtcheck vet test race fuzz verify spec-install spec-check demo-smoke demo showcase docs-check bench vuln clean

help:
	@printf '%s\n' 'build          Build bin/dither-mcp.' 'verify         Check formatting, code, docs, schemas, and MCP workflows.' 'palette-docs   Generate the palette reference from the engine catalog.' 'spec-install   Install the pinned OpenSpec and Quint development tools.' 'spec-check     Check OpenSpec specifications and Quint models.' 'fuzz           Run bounded engine, request, metadata, and profile fuzz tests.' 'vuln           Check dependencies with govulncheck.' 'showcase       Generate the original artwork and gallery.' 'demo           Record four VHS terminal demos.' 'bench          Run engine benchmarks.' 'clean          Remove only local binaries and coverage files.'
build:
	go build -trimpath -o bin/dither-mcp ./cmd/dither-mcp
fmt:
	gofmt -w cmd engine internal scripts
fmtcheck:
	@test -z "$$(gofmt -l cmd engine internal scripts)" || (gofmt -l cmd engine internal scripts; exit 1)
vet:
	go vet ./...
test:
	go test -timeout 2m ./...
race:
	go test -race -timeout 3m ./...
fuzz:
	go test ./engine -run '^$$' -fuzz FuzzProcess -fuzztime 5s -parallel 2
	go test ./internal/app -run '^$$' -fuzz FuzzStrictJSON -fuzztime 5s -parallel 2
	go test ./internal/imagemeta -run '^$$' -fuzz FuzzMetadata -fuzztime 5s -parallel 2
	go test ./internal/colorprofile -run '^$$' -fuzz FuzzICCProfile -fuzztime 5s -parallel 2
verify: fmtcheck vet test race build docs-check schemas-check palette-docs-check demo-smoke
spec-install:
	pnpm install --frozen-lockfile --ignore-scripts
spec-check:
	pnpm spec:check
demo-smoke: build
	go run ./scripts/mcp-demo
demo: build
	./scripts/showcase-record.sh
showcase:
	./scripts/showcase.sh
docs-check:
	go run ./scripts/check-docs
bench:
	go test ./engine -run '^$$' -bench . -benchmem
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
clean:
	rm -rf bin coverage.out

.PHONY: schemas schemas-check video-test
schemas:
	go run ./scripts/schema > docs/tool-schemas.json
schemas-check:
	@task_schema=$$(mktemp); trap 'rm -f "$$task_schema"' EXIT; go run ./scripts/schema > "$$task_schema"; cmp "$$task_schema" docs/tool-schemas.json
video-test:
	DITHER_TEST_VIDEO=1 go test -timeout 1m ./internal/app -run 'TestVideoIntegration|TestVideoInputContainers' -v

.PHONY: palette-docs palette-docs-check
palette-docs:
	go run ./scripts/palette-docs
palette-docs-check:
	go run ./scripts/palette-docs -check
