.PHONY: build test

build:
	go build -o errstats .

test: build
	./tests/run-all.sh
