FROM golang:1.22.4

WORKDIR /app

COPY . .

RUN go mod download

EXPOSE 3000

CMD ["go", "run", "main.go", "auth_controller.go"]