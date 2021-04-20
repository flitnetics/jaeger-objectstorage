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

	"github.com/go-pg/pg/v9"

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
       db                  *pg.DB
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
func NewWriter(db *pg.DB, cfg *types.Config, store lstore.Store, logger hclog.Logger) *Writer {
	w := &Writer{
		db: db,
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
               
        var labelsCustom = "{__name__=\"services\", env=\"prod\"}"
        existingChunks, err := w.store.Get(ctx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(labelsCustom)...)
        if err != nil {
                log.Println("error getting existing chunks %s", existingChunks)
        }
        //log.Println("existingChunks: %s", existingChunks)
	for _, tr := range chunksToBuildForTimeRanges {

                serviceChk := newChunk(buildTestStreams(labelsWithName, tr))
                if !contains(existingChunks, serviceChk) {
                        // service chunk
                        err := w.store.PutOne(ctx, serviceChk.From, serviceChk.Through, serviceChk)
                        // err := w.store.Put(ctx, []chunk.Chunk{chk})
                        if err != nil {
                                log.Println("store PutOne error: %s", err)
                        }
                        addedServicesChunkIDs[serviceChk.ExternalKey()] = struct{}{}
                }
	}

	return nil
}

func contains(s []chunk.Chunk, e chunk.Chunk) bool {
    for _, a := range s {
        if a.Metric[2].Value == e.Metric[2].Value {
            return true
        }
    }
    return false
}

func parseDate(in string) time.Time {
	t, err := time.Parse("2006-01-02", in)
	if err != nil {
		panic(err)
	}
	return t
}

func insertLogs(db *pg.DB, input *model.Span) (ret []*Log, err error) {
	ret = make([]*Log, 0, len(input.Logs))
	if input.Logs == nil {
		return ret, err
	}
	for _, log := range input.Logs {
		itm := &Log{SpanID: input.SpanID, Timestamp: log.Timestamp, Fields: mapModelKV(log.Fields)}
		ret = append(ret, itm)

		if _, err = db.Model(itm).Insert(); err != nil {
			return ret, err
		}
	}
	return ret, err
}

func insertRefs(db *pg.DB, input *model.Span) (ret []*SpanRef, err error) {
	ret = make([]*SpanRef, 0, len(input.References))
	if input.References == nil {
		return ret, err
	}
	for _, ref := range input.References {
		if ref.SpanID > 0 {
			itm := &SpanRef{SourceSpanID: input.SpanID, ChildSpanID: ref.SpanID, TraceIDLow: ref.TraceID.Low, TraceIDHigh: ref.TraceID.High, RefType: ref.RefType}
			ret = append(ret, itm)

			if _, err := db.Model(itm).Insert(); err != nil {
				return ret, err
			}
		}
	}
	return ret, err
}

/*func (w *Writer) batchAndWrite() {
	defer w.writeWG.Done()

	batch := make([]string, 0, common.MaxFlushPoints)
	var t <-chan time.Time

	for {
		select {
		case point, ok := <-w.writeCh:
			if !ok {
				if len(batch) > 0 {
					w.writeBatch(batch)
					return
				}
			}

			if t == nil {
				t = time.After(common.MaxFlushInterval)
			}

			batch = append(batch, point)

			if len(batch) == cap(batch) {
				//w.writeBatch(batch)
				batch = batch[:0]
				t = nil
			}

		case <-t:
			//w.writeBatch(batch)
			batch = batch[:0]
			t = nil
		}
	}
}
*/

