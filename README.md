This is the repository that contains S3 plugin for Jaeger.

> IMPORTANT: This plugin is still under development.

## Compile
In order to compile the plugin from source code you can use `go build`:

```
CGO_ENABLED=0 go build ./cmd/jaeger-s3/
```

## Start
In order to start plugin just tell jaeger the path to a config compiled plugin (password can be passed also as ENV: DB_PASSWORD).

```
jaeger-all-in-one --grpc-storage-plugin.binary=./jaeger-s3 --grpc-storage-plugin.configuration-file=./config-example.yaml
```

## License

The S3 Storage gRPC Plugin for Jaeger is an [MIT licensed](LICENSE) open source project.
