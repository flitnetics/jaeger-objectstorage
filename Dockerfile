FROM golang:1.17.3-alpine3.14 as BUILD
WORKDIR /build
COPY . .
RUN go build ./cmd/jaeger-objectstorage/

FROM alpine:3.14.2 as FINAL
COPY --from=BUILD /build/jaeger-objectstorage /go/bin/jaeger-objectstorage
RUN mkdir /plugin
# /plugin/ location is defined in jaeger-operator
CMD ["cp", "-r", "/go/bin/jaeger-objectstorage", "/plugin/jaeger-objectstorage"]
