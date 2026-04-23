build-chat:
	@go build -o ./bin/chat.exe ./...

run-chat: build-chat
	@ ./bin/chat.exe

test-chat-race:
	@go clean -testcache
	@go test -race -v ./...

test-chat:
	@go clean -testcache
	@go test -v ./...
