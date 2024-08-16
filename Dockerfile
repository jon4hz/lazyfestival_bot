FROM golang:1.22-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o lazyfestival_bot .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/lazyfestival_bot .
CMD ["./lazyfestival_bot"]
