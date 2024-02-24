This is the repository that contains object storage (S3/GCS/AzureBlob) plugin for Jaeger.

## About
S3, Google Cloud Storage(GCS) and Microsoft Azure Blob Storage object storage support for Jaeger. 

**Version 3 of this plugin is not compatible with Version 2**

## Build/Compile
In order to compile the plugin from source code you can use `go build`:

```
cd /path/to/jaeger-objectstorage
go build ./cmd/jaeger-objectstorage
```

## Configuration
#### Backend
```config.yaml
backend: your.tempo.backend:3200
```

## License

The Object Storage gRPC Plugin for Jaeger is an [Apache licensed](LICENSE) open source project.
