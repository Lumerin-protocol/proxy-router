FROM golang:1.25.4-bookworm as builder

# Capture the Git tag, commit hash, and architecture
ARG TAG_NAME
ARG COMMIT
ARG TARGETOS
ARG TARGETARCH
ENV TAG_NAME=$TAG_NAME
ENV COMMIT=$COMMIT

WORKDIR /app 
COPY . .

# Build the Go binary for the target platform
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 \
    TAG_NAME=$TAG_NAME COMMIT=$COMMIT ./build.sh && \
    cp /bin/sh /app/sh && chmod +x /app/sh

FROM scratch
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/proxy-router /usr/bin/
COPY --from=builder /app/sh /bin/sh

SHELL ["/bin/sh", "-c"]
EXPOSE 3333 8081

ENTRYPOINT ["proxy-router"]