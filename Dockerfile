FROM golang:1.22-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o greenfield_bot .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/greenfield_bot .
COPY ./data.json .
CMD ["./greenfield_bot"]
