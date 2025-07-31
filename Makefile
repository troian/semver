.PHONY: lint
lint:
	@echo "==> Linting codebase"
	@golangci-lint run

.PHONY: test
test:
	@echo "==> Running tests"
	GO111MODULE=on go test -v

.PHONY: test-cover
test-cover:
	@echo "==> Running Tests with coverage"
	GO111MODULE=on go test -cover .

.PHONY: fuzz
fuzz:
	@echo "==> Running Fuzz Tests"
	go env GOCACHE
	go test -fuzz=FuzzNewVersion -fuzztime=15s .
	go test -fuzz=FuzzStrictNewVersion -fuzztime=15s .
	go test -fuzz=FuzzNewConstraint -fuzztime=15s .
