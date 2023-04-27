FROM golang:1.19-buster as builder

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates make unzip && \
    rm -rf /var/lib/apt/lists/*
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
ADD ./docker/download_protoc /
ARG PB_VER="21.12"
RUN /download_protoc /usr/local

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
ENV PROTOC_BIN="/usr/local/bin/protoc"
RUN /usr/local/bin/protoc --version && \
    make $APP_NAME

FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ARG APP_NAME "node"

RUN printf '#!/bin/sh -ux\nexec /app/'$APP_NAME' $@\n' > /app-entrypoint && \
    chmod +x "/app-entrypoint"

COPY --from=builder /app/build/$APP_NAME /app/$APP_NAME

ENTRYPOINT ["/app-entrypoint"]
CMD ["start"]
