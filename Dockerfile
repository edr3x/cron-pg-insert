FROM golang:1.21.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM scratch

WORKDIR /app

COPY --from=builder /app/main .

COPY views/ ./views

EXPOSE 8080 

CMD ["./main"]
