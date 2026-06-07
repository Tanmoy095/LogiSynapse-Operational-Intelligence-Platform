FROM golang:1.24-alpine AS builder

WORKDIR /app/services/authentication-service
COPY services/authentication-service/go.mod services/authentication-service/go.sum ./
RUN go mod download
COPY services/authentication-service ./
RUN go build -o /out/authentication-service ./cmd

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /out/authentication-service /app/authentication-service
COPY services/authentication-service/db/migrations /app/db/migrations
EXPOSE 50052
CMD ["/app/authentication-service"]
