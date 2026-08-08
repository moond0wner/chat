include .env
export

run:
	@go mod tidy && go run cmd/tcp/main.go