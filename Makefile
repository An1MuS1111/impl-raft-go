.PHONY: build run clean test

# Configuration Variables
BINARY_NAME=impl-raft-go
MAIN_PATH=. 
OUTPUT_DIR=./bin
BINARY_PATH=$(OUTPUT_DIR)/$(BINARY_NAME)

# --- Targets ---

# Build the application binary
build:
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(BINARY_PATH) $(MAIN_PATH)

# Run the application (using 'go run' for simplicity, but you could run the built binary)
run:
	go run $(MAIN_PATH)/main.go

# Run all tests
test: 
	go test -v ./...

# Clean the build artifacts
.PHONY: clean
clean:
	rm -rf $(OUTPUT_DIR)


.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/$(target).proto

.PHONY: spawn
spawn: build
# NOTE: The line below must start with a literal TAB character.
	chmod +x spawn.sh
# NOTE: The line below must start with a literal TAB character.
	./spawn.sh