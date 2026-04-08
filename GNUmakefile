default: build

setup:
	git config core.hooksPath .githooks
	@echo "Git hooks configured."

build:
	go build -v ./...

install: build
	go install -v ./...

generate:
	cd tools && go generate ./...
	go generate ./...

test:
	go test -v -count=1 ./...

testacc:
	TF_ACC=1 go test -v -count=1 -timeout 120m ./internal/...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

.PHONY: setup build install generate test testacc lint fmt tidy
