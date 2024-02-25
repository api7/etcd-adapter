FROM golang:1.20 AS build-env

ARG ENABLE_PROXY=false

WORKDIR /build
COPY go.* ./

RUN go mod download

COPY . .
RUN make build

FROM ubuntu:latest

WORKDIR /etcd-adapter

COPY --from=build-env /build/etcd-adapter /etcd-adapter

COPY conf/ /etcd-adapter/conf/

EXPOSE 12379

ENTRYPOINT ["/etcd-adapter/etcd-adapter", "-c", "/etcd-adapter/conf/config.yaml"]
