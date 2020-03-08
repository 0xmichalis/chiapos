PWD = $(shell pwd)
TOOLS = tools
GOBIN = ${PWD}/${TOOLS}
PATH := ${GOBIN}:$(PATH)

bench:
	@go test -run=Bench -bench=. ./...
.PHONY: bench

build: build-binaries
	@go build ./...
.PHONY: build

build-binaries:
	@go build -o $(PWD)/bin/plotter  $(PWD)/cmd/plotter
	@go build -o $(PWD)/bin/prover   $(PWD)/cmd/prover
	@go build -o $(PWD)/bin/verifier $(PWD)/cmd/verifier
.PHONY: build-binaries

clean:
	@rm -rf $(PWD)/bin
	@rm -rf plot.dat
.PHONY: clean

test:
	@go test -count=1 -race -cover ./...
.PHONY: test

tools: ${TOOLS}/goimports
.PHONY: tools

${TOOLS}/goimports: go.sum
	@go build -o ${TOOLS}/goimports golang.org/x/tools/cmd/goimports

verify: ${TOOLS}/goimports
	@go vet ./...
	@./hack/verify_format.sh
.PHONY: verify
