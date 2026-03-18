# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /multi-git-mirror ./cmd/main.go

# Runtime stage
FROM alpine:3.23

RUN apk add --no-cache git git-lfs openssh-client

COPY --from=builder /multi-git-mirror /usr/local/bin/multi-git-mirror

ENTRYPOINT ["multi-git-mirror"]
