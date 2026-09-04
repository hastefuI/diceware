BINARY   := diceware
WORDLIST := wordlists/wordlist-basque-diceware.txt

.DEFAULT_GOAL := help
.PHONY: help build install run test

help:
	@echo "build    build $(BINARY)"
	@echo "install  install $(BINARY) into GOBIN"
	@echo "run      run the live view against $(WORDLIST)"
	@echo "test     go test ./..."

build:
	go build -trimpath -o $(BINARY) ./cmd/diceware

install:
	go install ./cmd/diceware

run:
	go run ./cmd/diceware -i $(WORDLIST)

test:
	go test ./...
