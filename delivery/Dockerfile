# Build
FROM golang:latest as builder
WORKDIR /go/src/app
COPY . .
RUN go-wrapper download
RUN CGO_ENABLED=0 GOOS=linux go-wrapper install

# Main image
FROM alpine
WORKDIR /go/bin
COPY --from=builder /go/bin/ .
CMD ["./app"]
