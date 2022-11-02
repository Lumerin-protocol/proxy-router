FROM golang:1.18-alpine as builder
WORKDIR /app 
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s -race" . && \
    cp /app/hashrouter /usr/bin 
# cp /bin/sh /app/sh && chmod +x /app/sh

# FROM scratch
# WORKDIR /app

# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# COPY --from=builder /app/hashrouter /usr/bin/
# COPY --chmod=0755 --from=builder /app/sh /bin/sh

# SHELL ["/bin/sh", "-c"]

ARG ETH_NODE_ADDRESS
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
ARG CONTRACT_MNEMONIC
ENV CONTRACT_MNEMONIC=$CONTRACT_MNEMONIC
ARG CLONE_FACTORY_ADDRESS
ENV CLONE_FACTORY_ADDRESS=$CLONE_FACTORY_ADDRESS
ARG WALLET_ADDRESS
ENV WALLET_ADDRESS=$WALLET_ADDRESS
ARG WALLET_PRIVATE_KEY
ENV WALLET_PRIVATE_KEY=$WALLET_PRIVATE_KEY
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
ARG VALIDATION_BUFFER_PERIOD=20m
ENV VALIDATION_BUFFER_PERIOD=$VALIDATION_BUFFER_PERIOD
ARG HASHRATE_DIFF_THRESHOLD
ENV HASHRATE_DIFF_THRESHOLD=$HASHRATE_DIFF_THRESHOLD

RUN echo "ETH_NODE_ADDRESS=$ETH_NODE_ADDRESS" \
    echo "WEB_ADDRESS=$WEB_ADDRESS" \
    echo "PROXY_ADDRESS=$PROXY_ADDRESS" \
    echo "CLONE_FACTORY_ADDRESS=$CLONE_FACTORY_ADDRESS" \
    echo "WALLET_ADDRESS=$WALLET_ADDRESS" \
    echo "IS_BUYER=$IS_BUYER" \
    echo "ENVIRONMENT=$ENVIRONMENT" \
    echo "PROXY_LOG_STRATUM=$PROXY_LOG_STRATUM" \
    echo "MINER_VETTING_PERIOD_SECONDS=$MINER_VETTING_PERIOD_SECONDS" \
    echo "MINER_VETTING_DURATION=$MINER_VETTING_DURATION" \
    echo "POOL_ADDRESS=$POOL_ADDRESS" \
    echo "POOL_MIN_DURATION=$POOL_MIN_DURATION" \
    echo "POOL_MAX_DURATION=$POOL_MAX_DURATION" \
    echo "ACCOUNT_INDEX=$ACCOUNT_INDEX" \
    echo "VALIDATION_BUFFER_PERIOD=$VALIDATION_BUFFER_PERIOD" \
    echo "HASHRATE_DIFF_THRESHOLD=$HASHRATE_DIFF_THRESHOLD"

EXPOSE 3333 8081

ENTRYPOINT ["hashrouter"]