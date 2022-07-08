install:
	go mod download

run: 
	go run main.go

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy