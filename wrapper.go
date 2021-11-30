package lumigotracer

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
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
func WrapHandler(handler interface{}, conf *Config) interface{} {
	if err := loadConfig(*conf); err != nil {
		logger.WithError(err).Error("failed validation error")
		return handler
	}
	if !cfg.debug {
		logger.Out = io.Discard
	}
	exporter, err := newExporter(cfg.PrintStdout)
	if err != nil {
		logger.WithError(err).Error("failed to create an exporter")
		return handler
	}
	ctx := context.Background()
	tracerProvider := trace.NewTracerProvider(
		trace.WithSpanProcessor(trace.NewBatchSpanProcessor(exporter)),
		trace.WithResource(newResource(ctx)),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func(ctx context.Context, payload json.RawMessage) (interface{}, error) {
		traceCtx, span := tracerProvider.Tracer("lumigo").Start(ctx, "LumigoParentSpan")
		defer span.End()

		response, lambdaErr := otellambda.WrapHandler(lambda.NewHandler(handler),
			otellambda.WithTracerProvider(tracerProvider),
			otellambda.WithFlusher(tracerProvider)).Invoke(traceCtx, payload)

		if data, err := json.Marshal(&payload); err == nil {
			span.SetAttributes(attribute.String("event", string(data)))
		} else {
			logger.WithError(err).Error("failed to track event")
		}

		if data, err := json.Marshal(json.RawMessage(response)); err == nil {
			span.SetAttributes(attribute.String("response", string(data)))
		} else {
			logger.WithError(err).Error("failed to track response")
		}

		if lambdaErr != nil {
			span.SetAttributes(attribute.String("exception", lambdaErr.Error()))
			return json.RawMessage(response), lambdaErr
		}
		tracerProvider.ForceFlush(ctx)
		return json.RawMessage(response), lambdaErr
	}
}

// WrapHandlerWithAWSConfig wraps the lambda handler passing AWS Config
func WrapHandlerWithAWSConfig(handler interface{}, cfg *Config, awsConfig *aws.Config) interface{} {
	TraceAWSClients(awsConfig)
	return WrapHandler(handler, cfg)
}

// newResource returns a resource describing this application.
func newResource(ctx context.Context) *resource.Resource {
	attrs := []attribute.KeyValue{
		attribute.String("lumigo_token", cfg.Token),
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
