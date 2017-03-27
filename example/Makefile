NY: build

GOFLAGS ?= $(GOFLAGS:) -o cperfc

all: install test

build:
	go build $(GOFLAGS) .

install:
	go get $(GOFLAGS) .

test: install
	go test $(GOFLAGS) .

bench: install
	go test -run=NONE -bench=. $(GOFLAGS) .

clean:
	go clean $(GOFLAGS) -i .


