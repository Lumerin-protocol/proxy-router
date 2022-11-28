install:
	go mod download

build:
	wire && go build
	
run:
	wire && go run .

test-unit:
	go test -v -p 1 -count 1 $(go list ./... | grep -v /test) 

test-e2e:
	go test -v -count=1 -timeout=30m -tags wireinject gitlab.com/TitanInd/hashrouter/test/e2e 

fmt:
	go fmt ./...

lint:
	go fmt ./...
	go vet ./...

tidy:
	go mod tidy