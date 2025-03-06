FROM golang:1.22.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .
COPY .env .
COPY views ./views
COPY articles.json .
COPY img ./img

RUN CGO_ENABLED=0 GOOS=linux go build -o servDev ./main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/servDev .
COPY --from=builder /app/.env .
COPY --from=builder /app/views ./views
COPY --from=builder /app/articles.json .
COPY --from=builder /app/img ./img

RUN apk add --no-cache ca-certificates

EXPOSE 8080

CMD ["./servDev"]