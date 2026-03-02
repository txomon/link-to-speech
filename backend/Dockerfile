FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o article-to-audio .

FROM alpine:3.21
RUN apk add --no-cache ffmpeg ca-certificates
COPY --from=builder /app/article-to-audio /usr/local/bin/
ENTRYPOINT ["article-to-audio"]
