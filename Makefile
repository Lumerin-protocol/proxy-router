install:
	go mod download

build:
	wire && go build
	
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