package lumigo

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var logger *log.Logger

func init() {
	logger = log.New()
	logger.Out = os.Stdout
	logger.Formatter = &log.JSONFormatter{}
}

// WrapHandler wraps the lambda handler
func WrapHandler(handler interface{}, cfg *Config) interface{} {

	exporter, err := newExporter(cfg.PrintStdout)
	if err != nil {
		return handler
	}
	ctx := context.Background()
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(newResource(ctx, cfg)),
	)

	otel.SetTracerProvider(tracerProvider)
	return otellambda.InstrumentHandler(handler,
		otellambda.WithTracerProvider(tracerProvider),
		otellambda.WithFlusher(tracerProvider))
}

// newResource returns a resource describing this application.
func newResource(ctx context.Context, cfg *Config) *resource.Resource {
	attrs := []attribute.KeyValue{
		attribute.String("lumigo_token", cfg.Token),
		attribute.String("service_name", cfg.ServiceName),
		semconv.ServiceNameKey.String(cfg.ServiceName),
	}
	if cfg.EnableThreadSafe {
		transactionID, _ := uuid.NewUUID()
		attrs = append(attrs, attribute.String("globalTransactionId", fmt.Sprintf("c_%s", transactionID.String())))
		parentID, _ := uuid.NewUUID()
		attrs = append(attrs, attribute.String("globalParentId", parentID.String()))
	}

	detector := lambdadetector.NewResourceDetector()
	res, err := detector.Detect(ctx)
	if err != nil {
		logger.WithError(err).Warn("failed to detect AWS lambda resources")
		return resource.NewWithAttributes(semconv.SchemaURL, attrs...)
	}
	r, _ := resource.Merge(
		res,
		resource.NewWithAttributes(res.SchemaURL(), attrs...),
	)
	return r
}

// newExporter returns a console exporter.
func newExporter(printStdout bool) (trace.SpanExporter, error) {
	if printStdout {
		return stdouttrace.New()
	}
	w, err := os.Create("/tmp/lumigo_tracing.json")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create data store")
	}

	return stdouttrace.New(
		stdouttrace.WithWriter(w),
	)
}
