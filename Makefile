BINARY_NAME=jira
BUILD_DIR=bin
MAIN_PKG=./cmd/jira

.PHONY: all build clean tidy run dev install uninstall

all: tidy build

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PKG)

clean:
	rm -rf $(BUILD_DIR)

tidy:
	go mod tidy

run: build
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

dev:
	air

install: build
	install -d /usr/local/bin
	install $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)
