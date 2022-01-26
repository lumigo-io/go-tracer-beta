package lumigotracer

import (
	"context"
	"encoding/json"
	"os"
	"reflect"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type tracer struct {
	provider *sdktrace.TracerProvider
	logger   logrus.FieldLogger
	span     trace.Span
	traceCtx context.Context
}

func init() {
	defer func() {
		if err := recover(); err != nil {
			logger.WithFields(logrus.Fields{
				"stacktrace": takeStacktrace(),
				"error":      err,
			}).Error("an exception occurred in lumigo's code")
		}
	}()
}

func NewTracer(ctx context.Context, provider *sdktrace.TracerProvider, logger logrus.FieldLogger) *tracer {
	traceCtx, span := provider.Tracer("lumigo").Start(ctx, "LumigoParentSpan")
	return &tracer{
		span:     span,
		traceCtx: traceCtx,
		provider: provider,
		logger:   logger,
	}
}

// Start tracks the span start data
func (t *tracer) Start(data []byte) {
	os.Setenv("IS_WARM_START", "true") // nolint
	t.span.SetAttributes(attribute.String("event", string(data)))
}

// Start tracks the span end data after lambda execution
func (t *tracer) End(response []byte, lambdaErr error) {
	if data, err := json.Marshal(json.RawMessage(response)); err == nil && lambdaErr == nil {
		t.span.SetAttributes(attribute.String("response", string(data)))
	} else {
		t.logger.WithError(err).Error("failed to track response")
	}

	if lambdaErr != nil {
		t.span.SetAttributes(attribute.String("error_type", reflect.TypeOf(lambdaErr).String()))
		t.span.SetAttributes(attribute.String("error_message", lambdaErr.Error()))
		t.span.SetAttributes(attribute.String("error_stacktrace", takeStacktrace()))
	}
	t.provider.ForceFlush(t.traceCtx)
	t.span.End()
}
