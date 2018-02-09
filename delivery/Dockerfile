# Build
FROM golang:latest as builder
WORKDIR /src
COPY . .
RUN go build -o main

# Main image
FROM alpine
WORKDIR /app
COPY --from=builder /src/main .
CMD ["./main"]
