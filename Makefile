
build:
	@go build ./...
.PHONY: build

test:
	@go test -count=1 -race -cover ./...
.PHONY: test
