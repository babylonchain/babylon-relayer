GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
DOCKER = $(shell which docker)

# Update changelog vars
ifneq (,$(SINCE_TAG))
	sinceTag := --since-tag $(SINCE_TAG)
endif
ifneq (,$(UPCOMING_TAG))
	upcomingTag := --future-release $(UPCOMING_TAG)
endif

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

update-changelog:
	@echo ./scripts/update_changelog.sh $(sinceTag) $(upcomingTag)
	./scripts/update_changelog.sh $(sinceTag) $(upcomingTag)

.PHONY: all build clean install test lint build-relayer-docker update-changelog
