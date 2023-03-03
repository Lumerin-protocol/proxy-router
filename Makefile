install:
	go mod download

build:
	wire && go build
	
run:
	wire && go run .

test-unit:
	go test -v -p 1 -count 1 $$(go list ./... | grep -v /test) 

fmt:
	go fmt ./...

lint:
	golangci-lint run -v

tidy:
	go mod tidy

clean:
	rm -rf ./logs ./hashrouter