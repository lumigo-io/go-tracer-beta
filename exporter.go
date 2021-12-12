package lumigotracer

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/lumigo-io/go-tracer/internal/telemetry"
	"github.com/lumigo-io/go-tracer/internal/transform"
	"github.com/pkg/errors"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Exporter exports OpenTelemetry data to New Relic.
type Exporter struct {
	startEncoder *json.Encoder
	endEncoder   *json.Encoder
	encoderMu    sync.Mutex

	stoppedMu sync.RWMutex
	stopped   bool
}

// New creates an Exporter with the passed options.
func NewExporter(startSpanWriter io.Writer, endSpanWriter io.Writer) (*Exporter, error) {
	startEnc := json.NewEncoder(startSpanWriter)
	endEnc := json.NewEncoder(endSpanWriter)
	return &Exporter{
		startEncoder: startEnc,
		endEncoder:   endEnc,
	}, nil
}

// ExportSpans writes spans in json format to file.
func (e *Exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if e == nil {
		return nil
	}

	e.stoppedMu.RLock()
	stopped := e.stopped
	e.stoppedMu.RUnlock()
	if stopped {
		return nil
	}

	if len(spans) == 0 {
		return nil
	}

	var lumigoSpans []telemetry.Span
	e.encoderMu.Lock()
	defer e.encoderMu.Unlock()
	for _, span := range spans {
		lumigoSpan := transform.Span(span)
		if span.Name() == os.Getenv("AWS_LAMBDA_FUNCTION_NAME") {
			if err := e.startEncoder.Encode([]telemetry.Span{lumigoSpan}); err != nil {
				return errors.Wrap(err, "failed to store startSpan")
			}
			continue
		}
		lumigoSpans = append(lumigoSpans, lumigoSpan)
	}

	if len(lumigoSpans) == 0 {
		return nil
	}
	err := e.endEncoder.Encode(lumigoSpans)
	if err != nil {
		return errors.Wrap(err, "failed to store endSpan")
	}
	return nil
}

// Shutdown is called to stop the exporter, it preforms no action.
func (e *Exporter) Shutdown(ctx context.Context) error {
	e.stoppedMu.Lock()
	e.stopped = true
	e.stoppedMu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil
}
