# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Dev stage for hot reloading
FROM golang:1.22-alpine AS dev
WORKDIR /app
RUN go install github.com/cosmtrek/air@v1.51.0
CMD ["air"]

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY db/migrations/ db/migrations/

EXPOSE 8080
CMD ["./server"]
