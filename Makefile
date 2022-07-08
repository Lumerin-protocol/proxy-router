install:
	go mod download

run:
	wire
	go run .

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy