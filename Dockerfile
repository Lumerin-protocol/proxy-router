FROM golang:1.18-alpine as builder

WORKDIR /app 
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" .

FROM scratch
WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/hashrouter /usr/bin/

EXPOSE 3333 8081

ARG ETH_NODE_ADDRESS="wss://ropsten.infura.io/ws/v3/91fa8dea25fe4bf4b8ce1c6be8bb9eb3" 
ENV ETH_NODE_ADDRESS=$ETH_NODE_ADDRESS
ARG WEB_ADDRESS 
ENV WEB_ADDRESS=$WEB_ADDRESS
ARG PROXY_ADDRESS 
ENV PROXY_ADDRESS=$PROXY_ADDRESS
ARG IS_BUYER 
ENV IS_BUYER=$IS_BUYER
ARG ENVIRONMENT 
ENV ENVIRONMENT=$ENVIRONMENT
ARG ACCOUNT_INDEX 
ENV ACCOUNT_INDEX=$ACCOUNT_INDEX
ARG PROXY_LOG_STRATUM 
ENV PROXY_LOG_STRATUM=$PROXY_LOG_STRATUM
ARG MINER_VETTING_PERIOD_SECONDS 
ENV MINER_VETTING_PERIOD_SECONDS=$MINER_VETTING_PERIOD_SECONDS
ARG MINER_VETTING_DURATION=1m #miner will not serve contracts for this period after connecing to proxy
ENV MINER_VETTING_DURATION=$MINER_VETTING_DURATION
ARG POOL_ADDRESS
ENV POOL_ADDRESS=$POOL_ADDRESS
ARG POOL_MIN_DURATION=2m # min online duration to get submits
ENV POOL_MIN_DURATION=$POOL_MIN_DURATION
ARG POOL_MAX_DURATION=5m
ENV POOL_MAX_DURATION=$POOL_MAX_DURATION

ENTRYPOINT ["hashrouter"]