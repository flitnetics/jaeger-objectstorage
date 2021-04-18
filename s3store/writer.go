package s3store

import (
	"io"
        "log"
        "io/ioutil"
        "path"
        "time"
        "context"
        "fmt"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"

        "github.com/weaveworks/common/user"

	pmodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"

	"github.com/grafana/loki/pkg/chunkenc"
	"github.com/grafana/loki/pkg/logql"
	"github.com/grafana/loki/pkg/logproto"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/cortexproject/cortex/pkg/chunk/storage"
	util_log "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/cortexproject/cortex/pkg/chunk"
	cortex_local "github.com/cortexproject/cortex/pkg/chunk/local"
	"github.com/cortexproject/cortex/pkg/ingester/client"

	"github.com/grafana/loki/pkg/util/validation"
 
        lstore "jaeger-s3/storage"
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
func NewWriter(db *pg.DB, cfg *types.Config, logger hclog.Logger) *Writer {
	w := &Writer{
		db: db,
                cfg: cfg,
		logger: logger,
	}

	db.CreateTable(&Service{}, &orm.CreateTableOptions{})
	db.CreateTable(&Operation{}, &orm.CreateTableOptions{})
	db.CreateTable(&Span{}, &orm.CreateTableOptions{})
	db.CreateTable(&SpanRef{}, &orm.CreateTableOptions{})
	db.CreateTable(&Log{}, &orm.CreateTableOptions{})

	return w
}

// Close triggers a graceful shutdown
func (w *Writer) Close() error {
	return nil
}

// WriteSpan saves the span into PostgreSQL
func (w *Writer) WriteSpan(span *model.Span) error {
	service := &Service{
		ServiceName: span.Process.ServiceName,
	}
	if _, err := w.db.Model(service).Where("service_name = ?", span.Process.ServiceName).
		OnConflict("(service_name) DO NOTHING").Returning("id").Limit(1).SelectOrInsert(); err != nil {
		return err
	}
	operation := &Operation{
		OperationName: span.OperationName,
	}
	if _, err := w.db.Model(operation).Where("operation_name = ?", span.OperationName).
		OnConflict("(operation_name) DO NOTHING").Returning("id").Limit(1).SelectOrInsert(); err != nil {
		return err
	}
        //log.Println("span data: %v" , span)
	if _, err := w.db.Model(&Span{
		ID:          span.SpanID,
		TraceIDLow:  span.TraceID.Low,
		TraceIDHigh: span.TraceID.High,
		OperationID: operation.ID,
		Flags:       span.Flags,
		StartTime:   span.StartTime,
		Duration:    span.Duration,
		Tags:        mapModelKV(span.Tags),
		ServiceID:   service.ID,
		ProcessID:   span.ProcessID,
		ProcessTags: mapModelKV(span.Process.Tags),
		Warnings:    span.Warnings,
	}).OnConflict("(id) DO UPDATE").Insert(); err != nil {
		return err
	}

	tempDir, err := ioutil.TempDir("", "boltdb-shippers")
        if err != nil {
                log.Println("tempDir failure %s", err)
        }

	limits, err := validation.NewOverrides(validation.Limits{}, nil)
        if err != nil {
                log.Println("limits failure %s", err)
        }

	// config for BoltDB Shipper
	boltdbShipperConfig := w.cfg.StorageConfig.BoltDBShipperConfig
	flagext.DefaultValues(&boltdbShipperConfig)

	// dates for activation of boltdb shippers
	storeDate := time.Now()

	kconfig := &lstore.Config{
		Config: storage.Config{
                        AWSStorageConfig: w.cfg.StorageConfig.AWSStorageConfig,
			FSConfig: cortex_local.FSConfig{Directory: path.Join(tempDir, "chunks")},
		},
		BoltDBShipperConfig: boltdbShipperConfig,
	}

        chunkStore, err := storage.NewStore(
		kconfig.Config,
                w.cfg.ChunkStoreConfig,
                w.cfg.SchemaConfig.SchemaConfig,
		limits,
                nil,
		nil,
		util_log.Logger,
	)

        if err != nil {
                log.Println("chunkStore error: %s", err)
        }

        log.Println("chunkStore: %s", chunkStore)

        var labelsWithName = fmt.Sprintf("{service_name=\"%s\", operation_name=\"%s\", __name__=\"zaihan6\", env=\"prod\"}",  span.Process.ServiceName, span.OperationName)

        if chunkStore != nil {
  	        store, err := lstore.NewStore(*kconfig, w.cfg.SchemaConfig, chunkStore, nil)
                if err != nil {
                       log.Println("store error: %s", err)
                }

	        // time ranges adding a chunk for each store and a chunk which overlaps both the stores
	        chunksToBuildForTimeRanges := []timeRange{
		        {
			        // chunk just for first store
			        storeDate,
			        storeDate.Add(span.Duration * time.Microsecond),
		        },
	        }

	        // build and add chunks to the store
	        addedServicesChunkIDs := map[string]struct{}{}
               
                existingChunks, err := store.Get(ctx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(labelsWithName)...)
	        for _, tr := range chunksToBuildForTimeRanges {

                        serviceChk := newChunk(buildTestStreams(labelsWithName, tr))
                        if !contains(existingChunks, serviceChk) {
                                // service chunk
                                err := store.PutOne(ctx, serviceChk.From, serviceChk.Through, serviceChk)
                                // err := store.Put(ctx, []chunk.Chunk{chk})
                                if err != nil {
                                        log.Println("store PutOne error: %s", err)
                                }
                                addedServicesChunkIDs[serviceChk.ExternalKey()] = struct{}{}
                        }
	        }
        }

	insertRefs(w.db, span)
	insertLogs(w.db, span)

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

