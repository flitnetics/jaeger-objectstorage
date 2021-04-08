package storage

import (
	"io"
        "context"

	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

var (
	_ spanstore.Writer = (*Writer)(nil)
	_ io.Closer        = (*Writer)(nil)
)

// Writer handles all writes to InfluxDB v1.x for the Jaeger data model
type Writer struct {
        test string
	logger hclog.Logger
}

func NewWriter(test string) *Writer {
	w := &Writer{
                test: test,
	}

	go func() {
		w.batchAndWrite()
	}()

	return w
}

// Close triggers a graceful shutdown
func (w *Writer) Close() error {
	return nil
}

// WriteSpan saves the span into Cassandra
func (w *Writer) WriteSpan(ctx context.Context, span *model.Span) error {
	return nil
}

func (w *Writer) batchAndWrite() {
}

func (w *Writer) writeBatch(batch []string) {
	w.logger.Warn("wrote points to InfluxDB", "quantity", len(batch))
}
