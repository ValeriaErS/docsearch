FROM golang:1.20

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o docsearch ./cmd/docsearch

EXPOSE 8080

CMD ["./docsearch", "web", "--port", ":8080"]