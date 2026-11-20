# Makefile — فناوریار
# دستورهای پرکاربرد توسعه و استقرار

BINARY := fanavaryar
WEBDIR := web

.PHONY: build web build-all test verify run clean

## Build backend binary
build:
	go build -o bin/$(BINARY) ./cmd/fanavaryar

## Install frontend dependencies and build production bundle
web:
	cd $(WEBDIR) && npm ci && npm run build

## Build backend + frontend
build-all: build web

## Run backend unit tests
test:
	go test ./...

## Run go vet + tests
verify:
	go vet ./... && go test ./...

## Start dev server (needs local PostgreSQL, see README)
run:
	go run ./cmd/fanavaryar

clean:
	rm -rf bin
	cd $(WEBDIR) && rm -rf dist node_modules 2>/dev/null || true
