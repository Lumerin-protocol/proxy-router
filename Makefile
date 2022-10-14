install:
	go mod download

run:
	wire && go run .

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	go fmt ./...
	go vet ./...

tidy:
	go mod tidy