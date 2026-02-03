FROM golang:1.25-alpine@sha256:d3f0cf7723f3429e3f9ed846243970b20a2de7bae6a5b66fc5914e228d831bbb AS build-env

WORKDIR /go/src/ts-fileserver

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

ARG VERSION_LONG
ENV VERSION_LONG=$VERSION_LONG

ARG VERSION_GIT
ENV VERSION_GIT=$VERSION_GIT

RUN go build -v -o ts-fileserver ./cmd/ts-fileserver

FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

RUN apk add --no-cache ca-certificates iptables iproute2 ip6tables

COPY --from=build-env /go/src/ts-fileserver/ts-fileserver /usr/local/bin

WORKDIR /data

ENTRYPOINT [ "/usr/local/bin/ts-fileserver" ]

