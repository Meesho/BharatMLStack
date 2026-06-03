.PHONY: build build-go build-rust \
        test test-go test-rust \
        coverage coverage-check \
        lint lint-go lint-rust \
        smoke-test \
        dev-etcd dev-stop dev-logs \
        clean

SCHEMA_DIR       := ./schema
CONTROLPLANE_DIR := ./controlplane
DATALOADER_DIR   := ./dataplane/dataloader
READSERVER_DIR   := ./dataplane/readserver
SDK_DIR          := ./sdk
SMOKE_TEST_DIR   := ./tools/smoke-test

GO_MODULES := $(SCHEMA_DIR) $(CONTROLPLANE_DIR) $(DATALOADER_DIR) $(SDK_DIR)

# ── Build ──────────────────────────────────────────────────────────────────────

build: build-go build-rust

build-go:
	@echo "Building Go modules..."
	@for dir in $(GO_MODULES); do \
		[ -f "$$dir/go.mod" ] && (echo "  $$dir" && cd "$$dir" && go build ./...) || true; \
	done

build-rust:
	@echo "Building Rust crate..."
	@[ -d "$(READSERVER_DIR)" ] && (cd "$(READSERVER_DIR)" && cargo build --release) || true

# ── Test ───────────────────────────────────────────────────────────────────────

test: test-go test-rust

test-go:
	@echo "Testing Go modules..."
	@for dir in $(GO_MODULES); do \
		[ -f "$$dir/go.mod" ] && (echo "  $$dir" && cd "$$dir" && go test -race ./...) || true; \
	done

test-rust:
	@echo "Testing Rust crate..."
	@[ -d "$(READSERVER_DIR)" ] && (cd "$(READSERVER_DIR)" && cargo test --verbose) || true

# ── Coverage ───────────────────────────────────────────────────────────────────

coverage:
	@echo "Coverage for all Go modules:"
	@for dir in $(GO_MODULES); do \
		[ -f "$$dir/go.mod" ] || continue; \
		echo "\n  --- $$dir ---"; \
		(cd "$$dir" && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out); \
	done

coverage-check:
	@echo "Enforcing 100% coverage..."
	@FAILED=0; \
	for dir in $(GO_MODULES); do \
		[ -f "$$dir/go.mod" ] || continue; \
		(cd "$$dir" && go test -race -coverprofile=coverage.out ./...) || { FAILED=1; continue; }; \
		TOTAL=$$(cd "$$dir" && go tool cover -func=coverage.out | grep '^total:' | awk '{print $$3}'); \
		echo "  $$dir: $$TOTAL"; \
		[ "$$TOTAL" = "100.0%" ] || { echo "  ERROR: $$dir is at $$TOTAL, want 100.0%"; FAILED=1; }; \
	done; \
	exit $$FAILED

# ── Lint ───────────────────────────────────────────────────────────────────────

lint: lint-go lint-rust

lint-go:
	@which staticcheck > /dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "Linting Go modules..."
	@for dir in $(GO_MODULES); do \
		[ -f "$$dir/go.mod" ] || continue; \
		echo "  $$dir"; \
		(cd "$$dir" && go vet ./... && staticcheck ./...); \
	done

lint-rust:
	@[ -d "$(READSERVER_DIR)" ] && \
		(cd "$(READSERVER_DIR)" && cargo fmt --all -- --check && cargo clippy --all-targets --all-features -- -D warnings) || true

# ── Smoke test ─────────────────────────────────────────────────────────────────

smoke-test:
	@[ -d "$(SMOKE_TEST_DIR)" ] && (cd "$(SMOKE_TEST_DIR)" && go run .) || \
		echo "smoke-test not yet available"

# ── Local dev ──────────────────────────────────────────────────────────────────

dev-etcd:
	docker compose -f quick-start/docker-compose.yml up -d
	@echo "etcd → localhost:2379"

dev-stop:
	docker compose -f quick-start/docker-compose.yml down

dev-logs:
	docker compose -f quick-start/docker-compose.yml logs -f

# ── Clean ──────────────────────────────────────────────────────────────────────

clean:
	@find . -name "coverage.out" -delete
	@find . -name "coverage.html" -delete
	@find . -name "*.sst" -delete
	@[ -d "$(READSERVER_DIR)" ] && (cd "$(READSERVER_DIR)" && cargo clean) || true
