BUILD_FOLDER  ?= build
BINARY_NAME   ?= near-delegation-watcher

.PHONY: build
build:
	@go build -o $(BUILD_FOLDER)/$(BINARY_NAME) 

.PHONY: test
test:
	@go clean -testcache && go test ./... -cover

.PHONY: run
run: 
	@go run main.go