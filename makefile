BUILD_FOLDER  ?= build
BINARY_NAME   ?= tezos-delegation-watcher
CONFIG_PATH   ?= config.yaml

.PHONY: build
build:
	@echo "Building the application..."
	@go build -o $(BUILD_FOLDER)/$(BINARY_NAME) 

.PHONY: test
test:
	@echo "Running tests..."
	@go clean -testcache && go test ./... -cover

.PHONY: run
run: build
	@echo "Running the application..."
	@CONFIG_PATH=$(CONFIG_PATH) ./$(BUILD_FOLDER)/$(BINARY_NAME)

.PHONY: clean	
clean:
	@echo "Cleaning up..."
	@go clean
	@rm -rf $(BUILD_FOLDER)

