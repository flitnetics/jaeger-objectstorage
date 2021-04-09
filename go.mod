module jaeger-s3

go 1.16

require (
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/cortexproject/cortex v1.7.1-0.20210224085859-66d6fb5b0d42
	github.com/fatih/color v1.9.0 // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-pg/pg/v9 v9.2.0
	github.com/gogo/googleapis v1.1.0 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20191106031601-ce3c9ade29de // indirect
	github.com/grafana/loki v1.6.1 // indirect
	github.com/hashicorp/go-hclog v0.15.0
	github.com/jaegertracing/jaeger v1.17.1
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/prometheus/common v0.18.0 // indirect
	github.com/prometheus/prometheus v1.8.2-0.20210215121130-6f488061dfb4 // indirect
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/viper v1.7.0
	github.com/thanos-io/thanos v0.13.1-0.20210226164558-03dace0a1aa1 // indirect
	github.com/weaveworks/common v0.0.0-20210112142934-23c8d7fa6120 // indirect
	go.etcd.io/bbolt v1.3.5-0.20200615073812-232d8fc87f50 // indirect
	google.golang.org/api v0.39.0 // indirect
	google.golang.org/grpc v1.37.0 // indirect
)

replace github.com/hpcloud/tail => github.com/grafana/tail v0.0.0-20201004203643-7aa4e4a91f03

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v36.2.0+incompatible

// Keeping this same as Cortex to avoid dependency issues.
replace k8s.io/client-go => k8s.io/client-go v0.19.4

replace k8s.io/api => k8s.io/api v0.19.4

replace github.com/hashicorp/consul => github.com/hashicorp/consul v1.5.1

// >v1.2.0 has some conflict with prometheus/alertmanager. Hence prevent the upgrade till it's fixed.
replace github.com/satori/go.uuid => github.com/satori/go.uuid v1.2.0

// Use fork of gocql that has gokit logs and Prometheus metrics.
replace github.com/gocql/gocql => github.com/grafana/gocql v0.0.0-20200605141915-ba5dc39ece85

// Same as Cortex, we can't upgrade to grpc 1.30.0 until go.etcd.io/etcd will support it.
replace google.golang.org/grpc => google.golang.org/grpc v1.29.1

// Same as Cortex
// Using a 3rd-party branch for custom dialer - see https://github.com/bradfitz/gomemcache/pull/86
replace github.com/bradfitz/gomemcache => github.com/themihai/gomemcache v0.0.0-20180902122335-24332e2d58ab

// Fix errors like too many arguments in call to "github.com/go-openapi/errors".Required
//   have (string, string)
//   want (string, string, interface {})
replace github.com/go-openapi/errors => github.com/go-openapi/errors v0.19.4

replace github.com/go-openapi/validate => github.com/go-openapi/validate v0.19.8
