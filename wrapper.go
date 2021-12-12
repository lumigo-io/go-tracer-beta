package lumigotracer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
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

const SPANS_DIR = "/tmp/lumigo-spans"
const SPAN_START_FILE = "/tmp/lumigo-spans/span_start"
const SPAN_END_FILE = "/tmp/lumigo-spans/span_end"

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
	exporter, err := newExporter(cfg.PrintStdout)
	if err != nil {
		logger.WithError(err).Error("failed to create an exporter")
		return handler
	}

	return func(ctx context.Context, payload json.RawMessage) (interface{}, error) {
		data, eventErr := json.Marshal(&payload)
		if eventErr != nil {
			logger.WithError(err).Error("failed to track event")
		}
		var tracerProvider *trace.TracerProvider
		if conf.tracerProvider == nil {
			tracerProvider = trace.NewTracerProvider(
				trace.WithSpanProcessor(trace.NewBatchSpanProcessor(exporter)),
				trace.WithResource(newResource(ctx,
					attribute.String("event", string(data)),
					attribute.String("tracer_version", version),
				)),
			)
		} else {
			tracerProvider = conf.tracerProvider
		}
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})

		traceCtx, span := tracerProvider.Tracer("lumigo").Start(ctx, "LumigoParentSpan")
		defer span.End()

		response, lambdaErr := otellambda.WrapHandler(lambda.NewHandler(handler),
			otellambda.WithTracerProvider(tracerProvider),
			otellambda.WithFlusher(tracerProvider)).Invoke(traceCtx, payload)

		os.Setenv("IS_COLD_START", "true") // nolint

		span.SetAttributes(attribute.String("tracer_version", version))
		if eventErr == nil {
			span.SetAttributes(attribute.String("event", string(data)))
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
		content, err := ioutil.ReadFile(SPAN_START_FILE)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(content))

		content, err = ioutil.ReadFile(SPAN_END_FILE)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(content))
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

// newExporter returns a console exporter.
func newExporter(printStdout bool) (trace.SpanExporter, error) {
	if printStdout {
		return stdouttrace.New()
	}
	if _, err := os.Stat(SPANS_DIR); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(SPANS_DIR, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
	startWriter, err := os.Create(SPAN_START_FILE)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create start data store")
	}
	endWriter, err := os.Create(SPAN_END_FILE)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create end data store")
	}

	return NewExporter(startWriter, endWriter)
}
