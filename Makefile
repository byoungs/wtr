.PHONY: build test install clean

BINARY := wtr
BUILD_DIR := bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/wtr

test:
	go test ./...

install: build
	cp $(BUILD_DIR)/$(BINARY) ~/go/bin/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)
