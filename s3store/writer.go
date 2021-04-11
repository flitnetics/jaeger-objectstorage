package s3store

import (
	"io"
        "log"
        _ "io/ioutil"
        _ "path"
        "time"
        "context"
        "flag"

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
	_ "github.com/cortexproject/cortex/pkg/chunk/storage"
	// util_log "github.com/cortexproject/cortex/pkg/util/log"
	"github.com/cortexproject/cortex/pkg/chunk"
	// cortex_local "github.com/cortexproject/cortex/pkg/chunk/local"
	"github.com/cortexproject/cortex/pkg/ingester/client"

	_ "github.com/grafana/loki/pkg/util/validation"
	_ "github.com/grafana/loki/pkg/cfg"
 
        lstore "jaeger-s3/storage"
        _ "jaeger-s3/storage/stores/shipper"

)

var _ spanstore.Writer = (*Writer)(nil)
var _ io.Closer = (*Writer)(nil)

var fooLabelsWithName = "{foo=\"bar\", __name__=\"logs\"}"

var (
	ctx        = user.InjectOrgID(context.Background(), "fake")
)

// Writer handles all writes to PostgreSQL 2.x for the Jaeger data model
type Writer struct {
       db                  *pg.DB
       spanMeasurement     string
       spanMetaMeasurement string
       logMeasurement      string

       logger hclog.Logger
}

type LConfig struct {
	StorageConfig    lstore.Config              `yaml:"storage_config,omitempty"`
	SchemaConfig     lstore.SchemaConfig        `yaml:"schema_config,omitempty"`
}

type MConfig struct {
	LConfig     `yaml:",inline"`
	configFile      string
}

type timeRange struct {
	from, to time.Time
}

func timeToModelTime(t time.Time) pmodel.Time {
	return pmodel.TimeFromUnixNano(t.UnixNano())
}

func buildTestStreams(labels string, tr timeRange) logproto.Stream {
        stream := logproto.Stream{
                Labels:  labels,
                Entries: []logproto.Entry{},
        }

        for from := tr.from; from.Before(tr.to); from = from.Add(time.Second) {
                stream.Entries = append(stream.Entries, logproto.Entry{
                        Timestamp: from,
                        Line:      from.String(),
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
        c := chunk.NewChunk("fake", client.Fingerprint(lbs), lbs, chunkenc.NewFacade(chk, 0, 0), from, through)
        // force the checksum creation
        if err := c.Encode(); err != nil {
                panic(err)
        }
        return c
}

func (c *LConfig) Validate() error {
	if err := c.SchemaConfig.Validate(); err != nil {
                log.Println("schema: %s", c.SchemaConfig)
                log.Println("invalid schema config")
	}
	if err := c.StorageConfig.Validate(); err != nil {
                log.Println("invalid storage config")
	}
	if err := c.StorageConfig.BoltDBShipperConfig.Validate(); err != nil {
                log.Println("invalid boltdb-shipper config")
	}
	return nil
}

// NewWriter returns a Writer for PostgreSQL v2.x
func NewWriter(db *pg.DB, logger hclog.Logger) *Writer {
	w := &Writer{
		db: db,
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

func (c *MConfig) RegisterFlags(f *flag.FlagSet) {
	f.StringVar(&c.configFile, "config.file", "/Users/zaihan/Projects/jaeger-s3/config-example.yaml", "yaml file to load")
	c.LConfig.LRegisterFlags(f)
}

func (c *LConfig) LRegisterFlags(f *flag.FlagSet) {
	c.StorageConfig.RegisterFlags(f)
	c.SchemaConfig.RegisterFlags(f)
}

func (c *MConfig) Clone() flagext.Registerer {
	return func(c MConfig) *MConfig {
		return &c
	}(*c)
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

        // https://github.com/grafana/loki/blob/306cc724380c1fca80948b4e29221a17bd9846b3/cmd/loki/main.go
	//var config MConfig

	/* if err := cfg.Parse(&config); err != nil {
                log.Println("failed to parse config %s", err)
	} 

	err := config.Validate()
	if err != nil {
                log.Println("failed to validate config %s", err)
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
	boltdbShipperConfig := shipper.Config{}
	flagext.DefaultValues(&boltdbShipperConfig)
	boltdbShipperConfig.ActiveIndexDirectory = path.Join(tempDir, "index")
	boltdbShipperConfig.SharedStoreType = "filesystem"
	boltdbShipperConfig.CacheLocation = path.Join(tempDir, "boltdb-shipper-cache")

	// dates for activation of boltdb shippers
	firstStoreDate := parseDate("2019-01-01")
	secondStoreDate := parseDate("2019-01-02")

	kconfig := lstore.Config{
		Config: storage.Config{
			FSConfig: cortex_local.FSConfig{Directory: path.Join(tempDir, "chunks")},
		},
		BoltDBShipperConfig: boltdbShipperConfig,
	}

	schemaConfig := lstore.SchemaConfig{
		chunk.SchemaConfig{
			Configs: []chunk.PeriodConfig{
				{
					From:       chunk.DayTime{Time: timeToModelTime(firstStoreDate)},
					IndexType:  "boltdb-shipper",
					ObjectType: "filesystem",
					Schema:     "v9",
					IndexTables: chunk.PeriodicTableConfig{
						Prefix: "index_",
						Period: time.Hour * 168,
					},
				},
				{
					From:       chunk.DayTime{Time: timeToModelTime(secondStoreDate)},
					IndexType:  "boltdb-shipper",
					ObjectType: "filesystem",
					Schema:     "v11",
					IndexTables: chunk.PeriodicTableConfig{
						Prefix: "index_",
						Period: time.Hour * 168,
					},
					RowShards: 2,
				},
			},
		},
	}

	lstore.RegisterCustomIndexClients(&config.LConfig.StorageConfig, nil)

	chunkStore, err := storage.NewStore(
		kconfig.Config,
		chunk.StoreConfig{},
		schemaConfig.SchemaConfig,
		limits,
		nil,
		nil,
		util_log.Logger,
	)
        if err != nil {
                log.Println("chunkStore error: %s", err)
        }

	store, err := lstore.NewStore(kconfig, schemaConfig, chunkStore, nil)
        if err != nil {
                log.Println("store error: %s", err)
        }

	// time ranges adding a chunk for each store and a chunk which overlaps both the stores
	chunksToBuildForTimeRanges := []timeRange{
		{
			// chunk just for first store
			secondStoreDate.Add(-3 * time.Hour),
			secondStoreDate.Add(-2 * time.Hour),
		},
		{
			// chunk overlapping both the stores
			secondStoreDate.Add(-time.Hour),
			secondStoreDate.Add(time.Hour),
		},
		{
			// chunk just for second store
			secondStoreDate.Add(2 * time.Hour),
			secondStoreDate.Add(3 * time.Hour),
		},
	}

	// build and add chunks to the store
	addedChunkIDs := map[string]struct{}{}
	for _, tr := range chunksToBuildForTimeRanges {
		chk := newChunk(buildTestStreams(fooLabelsWithName, tr))

		err := store.PutOne(ctx, chk.From, chk.Through, chk)
                if err != nil {
                        log.Println("store put error: %s", err)
                }

		addedChunkIDs[chk.ExternalKey()] = struct{}{}
	} */

	insertRefs(w.db, span)
	insertLogs(w.db, span)

	return nil
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

