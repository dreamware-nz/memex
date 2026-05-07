.PHONY: build test fmt vet
build:
	go build ./cmd/memex

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...
