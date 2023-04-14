# Use an official Golang runtime as a parent image

FROM golang:1.20 AS build-env

ARG ENABLE_PROXY=false

WORKDIR /build
COPY go.* ./

RUN if [ "$ENABLE_PROXY" = "true" ] ; then go env -w GOPROXY=https://goproxy.cn,direct ; fi \
    && go mod download

COPY . .
RUN make build

FROM ubuntu:latest

# Set the working directory to /go/src/app
WORKDIR /etcd-adapter

# Copy the current directory contents into the container at /go/src/app
COPY --from=build-env /build/etcd-adapter /etcd-adapter

COPY conf/ /etcd-adapter/conf/

# Expose port 8080 for the application
EXPOSE 12379

# Run the applicat/usr/local/etcd-adapterion when the container starts
ENTRYPOINT ["/etcd-adapter/etcd-adapter", "-c", "/etcd-adapter/conf/config.yaml"]
