This is the repository that contains object storage connector for Jaeger.

## About
Jaeger connector to Tempo

**Version 3 of this plugin is not compatible with Version 2**

## Build/Compile
In order to compile the plugin from source code you can use `go build`:

```
cd /path/to/jaeger-objectstorage
go build ./cmd/jaeger-objectstorage
```

## Configuration
### Requirements
Use our fork of tempo [https://github.com/flitnetics/tempo](HERE)

#### Backend
```config.yaml
backend: your.tempo.backend:3200
```

## License

The Object Storage gRPC Plugin for Jaeger is an [Apache licensed](LICENSE) open source project.
