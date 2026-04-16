FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o compiler cmd/main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/compiler .

RUN apk add --no-cache ca-certificates

EXPOSE 8080

CMD ["./compiler"]
