FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o mongo-bench

FROM ubuntu:jammy

WORKDIR /app

COPY --from=builder /app/mongo-bench /app/mongo-bench
COPY --from=builder /app/mongo-bench-entrypoint.sh /app/mongo-bench-entrypoint.sh

RUN apt-get update && \
    apt-get install -y netcat-openbsd wget gnupg jq && \
    wget -qO- https://www.mongodb.org/static/pgp/server-8.0.asc | tee /etc/apt/trusted.gpg.d/server-8.0.asc && \
    echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/8.0 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-8.0.list

RUN apt-get update && \
    apt-get install -y mongodb-mongosh

ENTRYPOINT ["/app/mongo-bench"]