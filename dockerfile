FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o docsearch ./cmd/docsearch

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/docsearch .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/web ./web

RUN mkdir -p /app/docs

EXPOSE 8080

CMD ["./docsearch", "serve", "--config", "configs/config.docker.yml"]