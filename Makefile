.PHONY: test test.all lint test.security test.arch test.smoke cover

test:
	go test ./domain/... ./application/... -coverprofile=coverage.out -race -v

test.all:
	go test ./...

lint:
	GOTOOLCHAIN=go1.26.0 golangci-lint run -v --timeout 12m

test.security:
	govulncheck ./...

test.arch:
	go test ./test/architecture/... -v

test.smoke:
	go test ./test/smoke/... -v

cover:
	go tool cover -html=coverage.out -o coverage.html
