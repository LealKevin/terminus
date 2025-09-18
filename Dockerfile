FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

WORKDIR /app/cmd/server
RUN go build -o main .

FROM alpine:latest AS runtime
WORKDIR /root
COPY --from=builder /app/cmd/server/main ./
EXPOSE 4200
CMD ["./main"]
