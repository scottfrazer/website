BINARY_NAME=api

build:
	go build -ldflags "-X main.GitHash=$(shell git rev-parse HEAD)" -o $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)