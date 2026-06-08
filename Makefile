.PHONY: server/build
server/build:
	@go build -o bin/server cmd/server/*.go

.PHONY: server/run
server/run: server/build
	@bin/server

.PHONY: sql/build
sql/build:
	@go tool sqlc generate
