FROM golang:1.18-alpine AS builder

RUN apk add ca-certificates alpine-sdk
WORKDIR /go/src/github.com/and3rson/raid
COPY go.mod go.sum ./
RUN go mod download -x
COPY cmd ./cmd
COPY raid ./raid
RUN mkdir /out && CGO_ENABLED=1 go build -o /out/raid ./cmd/raid/main.go

FROM alpine:3.15.4
# WORKDIR /etc/ssl/certs
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /root
COPY --from=builder /out/ /root/
ENTRYPOINT ["/root/raid"]
