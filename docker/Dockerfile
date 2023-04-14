FROM golang:1.19-buster as builder

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates make protobuf-compiler && \
    rm -rf /var/lib/apt/lists/* && \
    protoc --version && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.* ./
RUN go mod download

# Copy local code to the container image.
COPY . ./

ARG APP_NAME "node"

# Build the binary.
RUN make $APP_NAME

FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ARG APP_NAME "node"

RUN printf '#!/bin/sh -ux\nexec /app/'$APP_NAME' $@\n' > /app-entrypoint && \
    chmod +x "/app-entrypoint"

COPY --from=builder /app/build/$APP_NAME /app/$APP_NAME

CMD ["/app-entrypoint"]
