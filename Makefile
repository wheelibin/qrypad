.PHONY: testdb-pg-up testdb-pg-down testdb-mysql-up testdb-mysql-down format lint test test-cover update-golden integration-test integration-down

# Default test target: runs fast unit tests, no Docker required.
# For integration tests, run `make integration-test`.
test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

testdb-pg-up:
	cd test-db/postgres && docker compose up -d

testdb-pg-down:
	cd test-db/postgres && docker compose down -v

testdb-mysql-up:
	cd test-db/mysql && docker compose up -d

testdb-mysql-down:
	cd test-db/mysql && docker compose down -v

test-sqlite-up:
	sqlite3 test-db/sqlite/sqlite.db < test-db/sqlite/init.sql

integration-test:
	@echo "Starting test databases..."
	@cd test-db/postgres && docker compose up -d --wait
	@cd test-db/mysql && docker compose up -d --wait
	@$(MAKE) test-sqlite-up
	@echo "Running integration tests..."
	@if go test -tags=integration -count=1 ./internal/db/...; then \
		echo "Integration tests passed. Tearing down..."; \
		(cd test-db/postgres && docker compose down -v); \
		(cd test-db/mysql && docker compose down -v); \
	else \
		echo "Integration tests FAILED. Containers left running for debugging."; \
		echo "Run 'make integration-down' to clean up."; \
		exit 1; \
	fi

integration-down:
	@(cd test-db/postgres && docker compose down -v)
	@(cd test-db/mysql && docker compose down -v)

format: ## Applies standard formatting (spaces, indents, etc.)
	@echo "  >  Formatting source code"
	@go install tool github.com/segmentio/golines
	@go install tool golang.org/x/tools/cmd/goimports
	go fmt ./...; \
	find . -path './.go' -prune -o -path './.cache' -prune -o -path './.go-tools' -prune -o -name '*.go' -print | xargs golines --max-len=150 -w; \
	goimports -local github.com/wheelibin/qrypad -w $$(go list -f '{{.Dir}}' ./... | xargs -I {} find {} -maxdepth 1 -name "*.go") && \
	go mod tidy

lint:
	golangci-lint run ./...

update-golden:
	go test ./internal/component/... -update
