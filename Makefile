GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
DOCKER = $(shell which docker)

all: lint install

build:
	@echo "building babylon-relayer binary..."
	@go build -mod=readonly -o build/babylon-relayer main.go

clean:
	@echo "removing build/"
	@rm -rf ./build

install:
	@echo "installing babylon-relayer binary..."
	@go build -mod=readonly -o $(GOBIN)/babylon-relayer main.go

test:
	@go test -mod=readonly -race ./...

lint:
	@golangci-lint run
	@find . -name '*.go' -type f -not -path "*.git*" | xargs gofmt -d -s
	@go mod verify

build-relayer-docker:
	@make -C contrib/images babylon-relayer

build-ibcsim-gaia-docker:
	$(MAKE) -C contrib/images ibcsim-gaia

.PHONY: all build clean install test lint build-relayer-docker build-ibcsim-gaia-docker