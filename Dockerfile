# Build stage: compile statically for portability (CGO disabled).
FROM golang:1.25.5-alpine3.21 AS builder
ENV CGO_ENABLED=0
ARG VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath \
    -ldflags "-X github.com/hexiosec/asm-cloud-connector/internal/version.version=${VERSION}" \
    -o /bin/asm-cloud-connector ./cmd/connector

# Runtime stage: minimal image with CA certs and non-root user.
FROM alpine:3.23

RUN apk add --no-cache ca-certificates \
    && addgroup -S asm \
    && adduser -S asm -G asm

WORKDIR /app

COPY --from=builder /bin/asm-cloud-connector /app/asm-cloud-connector

USER asm

ENTRYPOINT ["/app/asm-cloud-connector"]
