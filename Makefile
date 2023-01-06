GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
DOCKER = $(shell which docker)

all: lint install

build: go.sum
ifeq ($(OS),Windows_NT)
	@echo "building babylon-relayer binary..."
	@go build -mod=readonly -o build/babylon-relayer.exe main.go
else
	@echo "building babylon-relayer binary..."
	@go build -mod=readonly -o build/babylon-relayer main.go
endif

clean:
	@echo "removing build/"
	@rm -rf ./build

install: go.sum
	@echo "installing babylon-relayer binary..."
	@go build -mod=readonly -o $(GOBIN)/babylon-relayer main.go

test:
	@go test -mod=readonly -race ./...

lint:
	@golangci-lint run
	@find . -name '*.go' -type f -not -path "*.git*" | xargs gofmt -d -s
	@go mod verify

relayer-docker: relayer-docker-rmi
	$(DOCKER) build --tag babylonchain/babylon-relayer .

relayer-docker-rmi:
	$(DOCKER) rmi babylonchain/babylon-relayer 2>/dev/null; true