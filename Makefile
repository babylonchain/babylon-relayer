GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin

all: lint install

build: go.sum
ifeq ($(OS),Windows_NT)
	@echo "building babylon-relayer binary..."
	@go build -mod=readonly -o build/babylon-relayer.exe main.go
else
	@echo "building babylon-relayer binary..."
	@go build -mod=readonly -o build/babylon-relayer main.go
endif

install: go.sum
	@echo "installing babylon-relayer binary..."
	@go build -mod=readonly -o $(GOBIN)/babylon-relayer main.go

test:
	@go test -mod=readonly -race ./...

lint:
	@golangci-lint run
	@find . -name '*.go' -type f -not -path "*.git*" | xargs gofmt -d -s
	@go mod verify
