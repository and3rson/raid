FROM golang:1.18-alpine AS builder

RUN apk add ca-certificates
WORKDIR /go/src/github.com/and3rson/raid
COPY *.go go.mod go.sum ./
RUN go get
COPY assets ./assets
COPY static ./static
RUN mkdir /out && CGO_ENABLED=0 go build -o /out/raid

FROM scratch
WORKDIR /etc/ssl/certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /root
COPY --from=builder /out/ /root/
ENTRYPOINT ["/root/raid"]
