FROM golang:alpine AS builder

RUN apk update && \
    apk add git build-base && \
    rm -rf /var/cache/apk/* && \
    mkdir -p "$GOPATH/src/github.com/buildsville/" && \
    git clone https://github.com/buildsville/service-target-pods-num-exporter.git && \
    mv service-target-pods-num-exporter "$GOPATH/src/github.com/buildsville/" && \
    cd "$GOPATH/src/github.com/buildsville/service-target-pods-num-exporter" && \
    GOOS=linux GOARCH=amd64 go build -o /service-target-pods-num-exporter

FROM alpine:3.7

RUN apk add --update ca-certificates

COPY --from=builder /service-target-pods-num-exporter /service-target-pods-num-exporter

ENTRYPOINT ["/service-target-pods-num-exporter"]
