package lumigotracer

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	lumigoctx "github.com/lumigo-io/go-tracer/internal/context"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const SPANS_DIR = "/tmp/lumigo-spans"

var logger *log.Logger

const (
	version = "0.1.0"
)

func init() {
	logger = log.New()
	logger.Out = os.Stdout
	logger.Formatter = &easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "#LUMIGO# - %time% - %lvl% - %msg%",
	}

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

	return func(ctx context.Context, payload json.RawMessage) (interface{}, error) {
		ctx = lumigoctx.NewContext(ctx, &lumigoctx.LumigoContext{
			TracerVersion: version,
		})

		exporter, err := createExporter(cfg.PrintStdout, ctx, logger)
		if err != nil {
			return lambda.NewHandler(handler).Invoke(ctx, payload)
		}

		data, eventErr := json.Marshal(&payload)
		if eventErr != nil {
			logger.WithError(err).Error("failed to track event")
		}
		var tracerProvider *trace.TracerProvider
		if conf.tracerProvider == nil {
			tracerProvider = trace.NewTracerProvider(
				trace.WithSyncer(exporter),
				trace.WithResource(newResource(ctx,
					attribute.String("event", string(data)),
				)),
			)
		} else {
			tracerProvider = conf.tracerProvider
		}
		otel.SetTracerProvider(tracerProvider)

		defer tracerProvider.ForceFlush(ctx)
		traceCtx, span := tracerProvider.Tracer("lumigo").Start(ctx, "LumigoParentSpan")
		defer span.End()

		response, lambdaErr := otellambda.WrapHandler(lambda.NewHandler(handler),
			otellambda.WithTracerProvider(tracerProvider),
			otellambda.WithFlusher(tracerProvider)).Invoke(traceCtx, payload)

		os.Setenv("IS_WARM_START", "true") // nolint

		if eventErr == nil {
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
		return json.RawMessage(response), lambdaErr
	}
}

// WrapHandlerWithAWSConfig wraps the lambda handler passing AWS Config
func WrapHandlerWithAWSConfig(handler interface{}, cfg *Config, awsConfig *aws.Config) interface{} {
	TraceAWSClients(awsConfig)
	return WrapHandler(handler, cfg)
}

// newResource returns a resource describing this application.
func newResource(ctx context.Context, extraAttrs ...attribute.KeyValue) *resource.Resource {
	attrs := []attribute.KeyValue{
		attribute.String("lumigo_token", cfg.Token),
	}
	attrs = append(attrs, extraAttrs...)
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

// createExporter returns a console exporter.
func createExporter(printStdout bool, ctx context.Context, logger log.FieldLogger) (trace.SpanExporter, error) {
	if printStdout {
		return stdouttrace.New()
	}
	if _, err := os.Stat(SPANS_DIR); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(SPANS_DIR, os.ModePerm); err != nil {
			return nil, errors.Wrapf(err, "failed to create dir: %s", SPANS_DIR)
		}
	} else if err != nil {
		logger.WithError(err).Error()
	}

	return newExporter(ctx, logger)
}
