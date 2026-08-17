.PHONY: fmt lint test build test-scenarios generate-protocol

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

lint:
	test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))"
	go vet ./...

test:
	go test ./...

build:
	go build ./...

test-scenarios:
	go test ./internal/mocknode -run 'TestNode_(Success|Reject|Timeout|Disconnect|Duplicate|LateResult|InvalidOffer|UnknownFields)$$'

generate-protocol:
	./scripts/generate-protocol.sh
