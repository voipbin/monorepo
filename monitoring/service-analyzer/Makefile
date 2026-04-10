.PHONY: build test install clean lint validate

BINARY := service-analyzer
GOFLAGS := -trimpath

build:
	go build $(GOFLAGS) -o $(BINARY) .

test:
	go test ./... -v

cover:
	go test -cover ./...

install:
	go install $(GOFLAGS) .

clean:
	rm -f $(BINARY)

lint:
	go vet ./...

validate: build
	./$(BINARY) validate --root ../..
