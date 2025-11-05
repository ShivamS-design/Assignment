.PHONY: test test-unit test-integration test-system test-security test-frontend bench clean

# Test commands
test: test-unit test-integration test-system test-security test-frontend

test-unit:
	@echo "Running unit tests..."
	cd backend && go test -v -race ./tests/unit/...
	cd rust-engine && cargo test --release
	cd frontend && npm run test:unit

test-integration:
	@echo "Running integration tests..."
	cd backend && go test -v -tags=integration ./tests/integration/...

test-system:
	@echo "Running system tests..."
	docker-compose -f docker-compose.test.yml up -d
	sleep 10
	cd backend && go test -v -tags=system ./tests/system/...
	docker-compose -f docker-compose.test.yml down

test-security:
	@echo "Running security tests..."
	cd backend && go test -v -tags=security ./tests/security/...
	gosec ./backend/...

test-frontend:
	@echo "Running frontend tests..."
	cd frontend && npm run test:unit && npm run test:e2e

bench:
	@echo "Running benchmarks..."
	cd backend && go test -bench=. -benchmem ./tests/... > benchmark-results.txt
	python3 scripts/check-performance-regression.py benchmark-results.txt

coverage:
	@echo "Generating coverage report..."
	cd backend && go test -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html

fuzz:
	@echo "Running fuzz tests..."
	cd rust-engine && cargo fuzz run wasm_parser -- -max_total_time=300
	cd backend && go test -fuzz=FuzzWasmParser -fuzztime=5m ./tests/security/

clean:
	rm -rf backend/coverage.out backend/coverage.html
	rm -rf frontend/coverage/
	rm -rf benchmark-results.txt performance-report.md
	docker-compose -f docker-compose.test.yml down -v

setup-test-env:
	@echo "Setting up test environment..."
	docker-compose -f docker-compose.test.yml up -d postgres redis
	sleep 5
	cd backend && go run scripts/migrate.go

lint:
	@echo "Running linters..."
	cd backend && golangci-lint run
	cd rust-engine && cargo clippy -- -D warnings
	cd frontend && npm run lint

security-scan:
	@echo "Running security scans..."
	gosec ./backend/...
	cargo audit --file rust-engine/Cargo.lock
	npm audit --prefix frontend/

install-tools:
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	cargo install cargo-audit cargo-fuzz