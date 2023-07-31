FROM golang:1.19.3-alpine as builder
WORKDIR /app 
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" . && \
    cp /app/hashrouter /usr/bin
# cp /bin/sh /app/sh && chmod +x /app/sh

# FROM scratch
# WORKDIR /app

# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# COPY --from=builder /app/hashrouter /usr/bin/
# COPY --chmod=0755 --from=builder /app/sh /bin/sh

# SHELL ["/bin/sh", "-c"]

ARG ACCOUNT_INDEX 
ARG CLONE_FACTORY_ADDRESS
ARG CONTRACT_HASHRATE_ADJUSTMENT
ARG CONTRACT_MNEMONIC
ARG ENVIRONMENT 
ARG ETH_NODE_ADDRESS
ARG HASHRATE_DIFF_THRESHOLD
ARG IS_BUYER 
ARG LOG_LEVEL
ARG MINER_SUBMIT_ERR_LIMIT
ARG MINER_VETTING_DURATION
ARG POOL_ADDRESS
ARG POOL_CONN_TIMEOUT
ARG POOL_MAX_DURATION
ARG POOL_MIN_DURATION
ARG PROXY_ADDRESS 
ARG PROXY_LOG_STRATUM 
ARG STRATUM_SOCKET_BUFFER_SIZE
ARG VALIDATION_BUFFER_PERIOD
ARG WALLET_ADDRESS
ARG WALLET_PRIVATE_KEY
ARG WEB_ADDRESS 
ARG WEB_PUBLIC_URL

ENV ACCOUNT_INDEX=$ACCOUNT_INDEX
ENV CLONE_FACTORY_ADDRESS=$CLONE_FACTORY_ADDRESS
ENV CONTRACT_HASHRATE_ADJUSTMENT=$CONTRACT_HASHRATE_ADJUSTMENT
ENV CONTRACT_MNEMONIC=$CONTRACT_MNEMONIC
ENV ENVIRONMENT=$ENVIRONMENT
ENV ETH_NODE_ADDRESS=$ETH_NODE_ADDRESS
ENV HASHRATE_DIFF_THRESHOLD=$HASHRATE_DIFF_THRESHOLD
ENV IS_BUYER=$IS_BUYER
ENV LOG_LEVEL=$LOG_LEVEL
ENV MINER_SUBMIT_ERR_LIMIT=$MINER_SUBMIT_ERR_LIMIT
ENV MINER_VETTING_DURATION=$MINER_VETTING_DURATION
ENV POOL_ADDRESS=$POOL_ADDRESS
ENV POOL_CONN_TIMEOUT=$POOL_CONN_TIMEOUT
ENV POOL_MAX_DURATION=$POOL_MAX_DURATION
ENV POOL_MIN_DURATION=$POOL_MIN_DURATION
ENV PROXY_ADDRESS=$PROXY_ADDRESS
ENV PROXY_LOG_STRATUM=$PROXY_LOG_STRATUM
ENV STRATUM_SOCKET_BUFFER_SIZE=$STRATUM_SOCKET_BUFFER_SIZE
ENV VALIDATION_BUFFER_PERIOD=$VALIDATION_BUFFER_PERIOD
ENV WALLET_ADDRESS=$WALLET_ADDRESS
ENV WALLET_PRIVATE_KEY=$WALLET_PRIVATE_KEY
ENV WEB_ADDRESS=$WEB_ADDRESS
ENV WEB_PUBLIC_URL=$WEB_PUBLIC_URL

RUN echo "ACCOUNT_INDEX=$ACCOUNT_INDEX" \
    echo "CLONE_FACTORY_ADDRESS=$CLONE_FACTORY_ADDRESS" \
    echo "CONTRACT_HASHRATE_ADJUSTMENT=$CONTRACT_HASHRATE_ADJUSTMENT" \
    echo "ENVIRONMENT=$ENVIRONMENT" \
    echo "HASHRATE_DIFF_THRESHOLD=$HASHRATE_DIFF_THRESHOLD" \
    echo "IS_BUYER=$IS_BUYER" \
    echo "LOG_LEVEL=$LOG_LEVEL" \
    echo "MINER_SUBMIT_ERR_LIMIT=$MINER_SUBMIT_ERR_LIMIT" \
    echo "MINER_VETTING_DURATION=$MINER_VETTING_DURATION" \
    echo "POOL_ADDRESS=$POOL_ADDRESS" \
    echo "POOL_CONN_TIMEOUT=$POOL_CONN_TIMEOUT" \
    echo "POOL_MAX_DURATION=$POOL_MAX_DURATION" \
    echo "POOL_MIN_DURATION=$POOL_MIN_DURATION" \
    echo "PROXY_ADDRESS=$PROXY_ADDRESS" \
    echo "PROXY_LOG_STRATUM=$PROXY_LOG_STRATUM" \
    echo "STRATUM_SOCKET_BUFFER_SIZE=$STRATUM_SOCKET_BUFFER_SIZE" \
    echo "VALIDATION_BUFFER_PERIOD=$VALIDATION_BUFFER_PERIOD" \
    echo "WALLET_ADDRESS=$WALLET_ADDRESS" \
    echo "WEB_ADDRESS=$WEB_ADDRESS" \
    echo "WEB_PUBLIC_URL=$WEB_PUBLIC_URL" \
    echo "ETH_NODE_ADDRESS=$ETH_NODE_ADDRESS"

EXPOSE 3333 8081

ENTRYPOINT ["hashrouter"]