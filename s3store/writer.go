package s3store

import (
        "context"
	"io"
        "time"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"

        "github.com/cortexproject/cortex/pkg/chunk"
        "github.com/cortexproject/cortex/pkg/chunk/local"
        "github.com/cortexproject/cortex/pkg/chunk/storage"
        util_log "github.com/cortexproject/cortex/pkg/util/log"

        "github.com/prometheus/client_golang/prometheus"
        pmodel "github.com/prometheus/common/model"

        "github.com/weaveworks/common/user"

        lstore "jaeger-s3/storage"
        "github.com/grafana/loki/pkg/util/validation"
)

var _ spanstore.Writer = (*Writer)(nil)
var _ io.Closer = (*Writer)(nil)
var (
        start     = pmodel.Time(1523750400000)
        ctx       = user.InjectOrgID(context.Background(), "fake")
        maxChunks = 1200 // 1200 chunks is 2gib ish of data enough to run benchmark
)

// Writer handles all writes to PostgreSQL 2.x for the Jaeger data model
type Writer struct {
	db                  *pg.DB
	spanMeasurement     string
	spanMetaMeasurement string
	logMeasurement      string

	// Points as line protocol
	//writeCh chan string
	//writeWG sync.WaitGroup

	logger hclog.Logger
}

func getStore() (lstore.Store, error) {
        storeConfig := lstore.Config{
                Config: storage.Config{
                        BoltDBConfig: local.BoltDBConfig{Directory: "/tmp/benchmark/index"},
                        FSConfig:     local.FSConfig{Directory: "/tmp/benchmark/chunks"},
                },
        }

        schemaCfg := lstore.SchemaConfig{
                SchemaConfig: chunk.SchemaConfig{
                        Configs: []chunk.PeriodConfig{
                                {
                                        From:       chunk.DayTime{Time: start},
                                        IndexType:  "boltdb",
                                        ObjectType: "filesystem",
                                        Schema:     "v9",
                                        IndexTables: chunk.PeriodicTableConfig{
                                                Prefix: "index_",
                                                Period: time.Hour * 168,
                                        },
                                },
                        },
                },
        }

        chunkStore, err := storage.NewStore(
                storeConfig.Config,
                chunk.StoreConfig{},
                schemaCfg.SchemaConfig,
                &validation.Overrides{},
                prometheus.DefaultRegisterer,
                nil,
                util_log.Logger,
        )
        if err != nil {
                return nil, err
        }

        return lstore.NewStore(storeConfig, schemaCfg, chunkStore, prometheus.DefaultRegisterer)
}

// NewWriter returns a Writer for PostgreSQL v2.x
func NewWriter(db *pg.DB, logger hclog.Logger) *Writer {
	w := &Writer{
		db: db,
		//writeCh: make(chan string),
		logger: logger,
	}

	db.CreateTable(&Service{}, &orm.CreateTableOptions{})
	db.CreateTable(&Operation{}, &orm.CreateTableOptions{})
	db.CreateTable(&Span{}, &orm.CreateTableOptions{})
	db.CreateTable(&SpanRef{}, &orm.CreateTableOptions{})
	db.CreateTable(&Log{}, &orm.CreateTableOptions{})

	//w.writeWG.Add(1)
	//go w.batchAndWrite()

	return w
}

// Close triggers a graceful shutdown
func (w *Writer) Close() error {
	//close(w.writeCh)
	//w.writeWG.Wait()
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

	insertRefs(w.db, span)
	insertLogs(w.db, span)

	return nil
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
