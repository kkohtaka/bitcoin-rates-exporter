# Copyright (C) 2018 Kazumasa Kohtaka <kkohtaka@gmail.com> All right reserved
# This file is available under the MIT license.

FROM golang:1.10.0 as builder
ENV GOPATH=/go
WORKDIR /go/src/github.com/kkohtaka/bitcoin-rates-exporter
COPY main.go .
COPY vendor ./vendor
RUN CGO_ENABLED=0 GOOS=linux go build -a -o bitcoin-rates-exporter

FROM alpine:3.7 as certs-installer
RUN apk add --update ca-certificates

FROM scratch
COPY --from=builder /go/src/github.com/kkohtaka/bitcoin-rates-exporter/bitcoin-rates-exporter /bin/bitcoin-rates-exporter
COPY --from=certs-installer /etc/ssl/certs /etc/ssl/certs
ENTRYPOINT ["/bin/bitcoin-rates-exporter"]
CMD [""]
