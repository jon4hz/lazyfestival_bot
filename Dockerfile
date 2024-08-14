FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o lazyfestival_bot .

FROM alpine:latest
RUN apk update && \
    apk add --no-cache tzdata
WORKDIR /app
COPY --from=builder /app/lazyfestival_bot .
CMD ["./lazyfestival_bot"]
