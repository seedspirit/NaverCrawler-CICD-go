FROM golang:1.20.4-alpine3.17 AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

COPY go.mod go.sum main.go ./
RUN go mod download

COPY . .

RUN go build -o main

FROM chromedp/headless-shell:113.0.5672.93

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /app/main .

ENTRYPOINT [ "./main" ]