BINARY_NAME=redmine-tui

.PHONY: all build clean run sync

all: build

build:
	go build -o $(BINARY_NAME) -v .

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f redmine_issues.html

run: build
	./$(BINARY_NAME)

sync: build
	./$(BINARY_NAME) --sync
