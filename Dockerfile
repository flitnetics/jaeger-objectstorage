FROM alpine:3.14.2

ADD jaeger-s3 /go/bin/jaeger-s3

RUN mkdir /plugin

# /plugin/ location is defined in jaeger-operator
CMD ["cp", "/go/bin/jaeger-s3", "/plugin/jaeger-s3"]
