# Dynamic Builds
ARG BUILDER_IMAGE=golang:1.20-buster
ARG FINAL_IMAGE=debian:buster-slim

# Build Stage
FROM ${BUILDER_IMAGE} AS builder

# Build Args
ARG GIT_REVISION=""

# Ensure ca-certificates are up to date on the image
RUN update-ca-certificates

# Use modules for dependencies
WORKDIR $GOPATH/src/github.com/rotationalio/ensign-benchmarks

COPY go.mod .
COPY go.sum .

ENV CGO_ENABLED=0
ENV GO111MODULE=on
RUN go mod download
RUN go mod verify

# Copy package
COPY . .

# Build binary
RUN go build -v -o /go/bin/enbench -ldflags="-X 'github.com/rotationalio/ensign-benchmarks/pkg.GitVersion=${GIT_REVISION}'" ./cmd/enbench

# Final Stage
FROM ${FINAL_IMAGE} AS final

LABEL maintainer="Rotational Labs <support@rotational.io>"
LABEL description="Benchmarking utilities for the Ensign eventing system"

# Ensure ca-certificates are up to date
RUN set -x && apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage
COPY --from=builder /go/bin/enbench /usr/local/bin/enbench

CMD [ "/usr/local/bin/enbench", "blast" ]