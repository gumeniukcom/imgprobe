all: fmt lint test

GOFILES = ./...

fmt:
	go fmt $(GOFILES)

lint:
	golangci-lint run $(GOFILES)

test:
	go test $(GOFILES)

testv:
	go test -v $(GOFILES)

testrace:
	go test -race $(GOFILES)

# Assert the package builds with cgo disabled — a headline guarantee.
build-nocgo:
	CGO_ENABLED=0 go build $(GOFILES)

bench:
	go test -run xxx -bench=. -benchmem $(GOFILES)

cover:
	go test -race -coverprofile=coverage.out -covermode=atomic $(GOFILES)
	go tool cover -func=coverage.out | tail -1

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 $(GOFILES)
