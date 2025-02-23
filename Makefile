# Makefile for age-plgin-op
#
# Copied from https://github.com/Foxboron/age-plugin-tpm/blob/master/Makefile

all: build

build: age-plugin-1p

age-plugin-1p:
	mkdir -p dist
	go build -o dist/age-plugin-1p ./cmd/age-plugin-1p

.PHONY: age-plugin-1p

test:
	go test ./...

check:
	staticcheck ./...
	go vet ./...

.PHONY: test
