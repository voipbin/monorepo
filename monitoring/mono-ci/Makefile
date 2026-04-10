.PHONY: build test cover install clean lint validate

build:
	go build -o mono-ci .

test:
	go test -v -count=1 ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@rm -f coverage.out

install:
	go install .

clean:
	rm -f mono-ci coverage.out

lint:
	go vet ./...

validate: lint test
	@echo "All checks passed."
