.PHONY: all build vet test test-verbose test-integration test-realdb docker-up docker-down verify coverage clean

GO ?= go

all: vet test

build:
	$(GO) build ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./... -count=1 -tags=integration

test-verbose:
	$(GO) test ./... -v -count=1 -tags=integration

# sqlmock-based integration tests (no real DB needed, build tag: integration)
# Run from root — db_integration_test.go has the integration build tag.
test-integration:
	$(GO) test -count=1 -tags=integration -v -run Integration

# real database tests — set XDB_TEST_DRIVER=sqlite3 (default) or postgres
test-realdb:
	$(GO) test ./tests/ -count=1 -tags=realdb -v

verify:
	$(GO) test ./... -count=1 -tags=integration -v

coverage:
	$(GO) test ./... -coverprofile=coverage.out -tags=integration
	$(GO) tool cover -html=coverage.out

# Docker Compose helpers for PostgreSQL (compose file in tests/)
docker-up:
	docker compose -f tests/docker-compose.yml up -d --wait

docker-down:
	docker compose -f tests/docker-compose.yml down

release:
	goreleaser release --clean

clean:
	rm -f coverage.out
