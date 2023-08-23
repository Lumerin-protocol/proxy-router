run:
	go run cmd/main.go

build:
	go build -ldflags="-s -w" -o bin/hashrouter cmd/main.go 

clean:
	rm -rf bin logs

install:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.53.3
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/praetorian-inc/gokart@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	
lint:
	golangci-lint run
	govulncheck ./...
	gokart scan .
	gosec ./...
