FROM golang:1.18-alpine as builder

WORKDIR /app 
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" .

FROM scratch
WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/hashrouter /usr/bin/

EXPOSE 3333 8081

ARG eth_node_address="wss://ropsten.infura.io/ws/v3/91fa8dea25fe4bf4b8ce1c6be8bb9eb3" # default value
ENV ETH_NODE_ADDRESS=$eth_node_address

ENTRYPOINT ["hashrouter"]