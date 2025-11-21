.PHONY: build run clean install

build:
	go build -o rtt-cli .

run: build
	./rtt-cli

clean:
	rm -f rtt-cli

install:
	go install

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...
