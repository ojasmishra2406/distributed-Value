.PHONY: all build test clean

all: build test

build:
	mkdir -p bin
	go build -o bin/node ./cmd/node
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/bench ./cmd/bench
	go build -o bin/kv-cli ./cmd/cli

test:
	go test -v -race ./...

clean:
	rm -rf bin/
	rm -rf data/
	rm -f *.wal
	rm -f *.sst
