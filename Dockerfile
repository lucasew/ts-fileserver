FROM golang:1.25-alpine AS build-env

WORKDIR /go/src/ts-fileserver

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

ARG VERSION_LONG
ENV VERSION_LONG=$VERSION_LONG

ARG VERSION_GIT
ENV VERSION_GIT=$VERSION_GIT

RUN go build -v -o ts-fileserver ./cmd/ts-fileserver

FROM alpine:3.22

RUN apk add --no-cache ca-certificates iptables iproute2 ip6tables

COPY --from=build-env /go/src/ts-fileserver/ts-fileserver /usr/local/bin

WORKDIR /data

ENTRYPOINT [ "/usr/local/bin/ts-fileserver" ]

