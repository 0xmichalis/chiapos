
build: build-binaries
	@go build ./...
.PHONY: build

build-binaries:
	@go build -o $(PWD)/bin/plotter $(PWD)/cmd/plotter
.PHONY: build-binaries

clean:
	@rm -rf $(PWD)/bin
.PHONY: clean

test:
	@go test -count=1 -race -cover ./...
.PHONY: test
