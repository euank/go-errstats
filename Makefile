.PHONY: build test

build:
	go build -o errstats .

test: build
	go test ./...
	./tests/run-all.sh
