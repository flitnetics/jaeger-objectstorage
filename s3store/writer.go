package s3store

import (
	"io"
        "log"
        "time"
        "context"
        "fmt"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"

        "github.com/weaveworks/common/user"

	pmodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"

	"github.com/grafana/loki/pkg/chunkenc"
	"github.com/grafana/loki/pkg/logql"
	"github.com/grafana/loki/pkg/logproto"

	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/ingester/client"

        lstore "github.com/grafana/loki/pkg/storage"
        "jaeger-s3/config/types"
)

var _ spanstore.Writer = (*Writer)(nil)
var _ io.Closer = (*Writer)(nil)

//var fooLabelsWithName = "{foo=\"bar\", __name__=\"logs\"}"

var (
	ctx        = user.InjectOrgID(context.Background(), "data")
)

// Writer handles all writes to PostgreSQL 2.x for the Jaeger data model
type Writer struct {
       spanMeasurement     string
       spanMetaMeasurement string
       logMeasurement      string

       cfg    *types.Config
       store  lstore.Store
       logger hclog.Logger
}

type timeRange struct {
	from, to time.Time
}

func buildTestStreams(labels string, tr timeRange) logproto.Stream {
        stream := logproto.Stream{
                Labels:  labels,
                Entries: []logproto.Entry{},
        }

        for from := tr.from; from.Before(tr.to); from = from.Add(time.Second) {
                stream.Entries = append(stream.Entries, logproto.Entry{
                        Timestamp: from,
                        Line:      "Hello there! I'm Jack Sparrow",
                })
        }

        return stream
}

func newChunk(stream logproto.Stream) chunk.Chunk {
        lbs, err := logql.ParseLabels(stream.Labels)
        if err != nil {
                panic(err)
        }
        if !lbs.Has(labels.MetricName) {
                builder := labels.NewBuilder(lbs)
                builder.Set(labels.MetricName, "logs")
                lbs = builder.Labels()
        }
        from, through := pmodel.TimeFromUnixNano(stream.Entries[0].Timestamp.UnixNano()), pmodel.TimeFromUnixNano(stream.Entries[0].Timestamp.UnixNano())
        chk := chunkenc.NewMemChunk(chunkenc.EncGZIP, 256*1024, 0)
        for _, e := range stream.Entries {
                if e.Timestamp.UnixNano() < from.UnixNano() {
                        from = pmodel.TimeFromUnixNano(e.Timestamp.UnixNano())
                }
                if e.Timestamp.UnixNano() > through.UnixNano() {
                        through = pmodel.TimeFromUnixNano(e.Timestamp.UnixNano())
                }
                _ = chk.Append(&e)
        }
        chk.Close()
        c := chunk.NewChunk("data", client.Fingerprint(lbs), lbs, chunkenc.NewFacade(chk, 0, 0), from, through)
        // force the checksum creation
        if err := c.Encode(); err != nil {
                panic(err)
        }
        return c
}

// NewWriter returns a Writer for PostgreSQL v2.x
func NewWriter(cfg *types.Config, store lstore.Store, logger hclog.Logger) *Writer {
	w := &Writer{
                cfg: cfg,
                store: store,
		logger: logger,
	}

	return w
}

// Close triggers a graceful shutdown
func (w *Writer) Close() error {
	return nil
}

// WriteSpan saves the span into PostgreSQL
func (w *Writer) WriteSpan(span *model.Span) error {

        var labelsWithName = fmt.Sprintf("{__name__=\"services\", env=\"prod\", id=\"%d\", trace_id_low=\"%d\", trace_id_high=\"%d\", flags=\"%d\", duration=\"%d\", tags=\"%s\", process_id=\"%s\", process_tags=\"%s\", warnings=\"%s\", service_name=\"%s\", operation_name=\"%s\"}",
        span.SpanID,
        span.TraceID.Low,
        span.TraceID.High,
        span.Flags,
        span.Duration,
        mapModelKV(span.Tags),
        span.ProcessID,
        mapModelKV(span.Process.Tags),
        span.Warnings,
        span.Process.ServiceName,
        span.OperationName)

        storeDate := time.Now()

        // time ranges adding a chunk for each store and a chunk which overlaps both the stores
        chunksToBuildForTimeRanges := []timeRange{
	        {
		        // chunk just for first store
		        storeDate,
		        storeDate.Add(span.Duration * time.Microsecond),
	        },
        }

        addedServicesChunkIDs := map[string]struct{}{}
	for _, tr := range chunksToBuildForTimeRanges {

                serviceChk := newChunk(buildTestStreams(labelsWithName, tr))
                // service chunk
                err := w.store.PutOne(ctx, serviceChk.From, serviceChk.Through, serviceChk)
                // err := w.store.Put(ctx, []chunk.Chunk{chk})
                if err != nil {
                        log.Println("store PutOne error: %s", err)
                }
                addedServicesChunkIDs[serviceChk.ExternalKey()] = struct{}{}
	}

	return nil
}

func parseDate(in string) time.Time {
	t, err := time.Parse("2006-01-02", in)
	if err != nil {
		panic(err)
	}
	return t
}
