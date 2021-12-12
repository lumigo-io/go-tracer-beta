package lumigotracer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/lumigo-io/go-tracer/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type exporterTestSuite struct {
	suite.Suite
}

func TestSetupExporterSuite(t *testing.T) {
	suite.Run(t, &exporterTestSuite{})
}

func (conf *exporterTestSuite) AfterTest() {
	_ = os.Remove(SPAN_START_FILE)
	_ = os.Remove(SPAN_END_FILE)
}

func (e *exporterTestSuite) TestNilExporter() {
	span := &tracetest.SpanStub{}
	var exporter *Exporter

	exporter.ExportSpans(context.Background(), []trace.ReadOnlySpan{span.Snapshot()}) //nolint
}

func (e *exporterTestSuite) TestExportSpans() {
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test")
	spanID, _ := oteltrace.SpanIDFromHex("83887e5d7da921ba")
	traceID, _ := oteltrace.TraceIDFromHex("83887e5d7da921ba")

	spanCtx := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		SpanID:  spanID,
		TraceID: traceID,
	})
	startSpan := &tracetest.SpanStub{
		Name:        "test",
		StartTime:   time.Now(),
		EndTime:     time.Now(),
		SpanContext: spanCtx,
		Attributes: []attribute.KeyValue{
			attribute.String("faas.execution", "3f12bdd4-651f-4610-a469-a797721cd438"),
			attribute.String("cloud.account.id", "123"),
		},
		Resource: resource.NewWithAttributes(semconv.SchemaURL,
			attribute.KeyValue{
				Key:   "cloud.provider",
				Value: attribute.StringValue("aws"),
			},
			attribute.KeyValue{
				Key:   "faas.name",
				Value: attribute.StringValue("test"),
			},
			attribute.KeyValue{
				Key:   "faas.name",
				Value: attribute.StringValue("test"),
			},
			attribute.KeyValue{
				Key:   "lumigo_token",
				Value: attribute.StringValue("test"),
			},
			attribute.KeyValue{
				Key:   "cloud.region",
				Value: attribute.StringValue("us-east-1"),
			},
		),
	}
	endSpan := &tracetest.SpanStub{
		Name:        "LumigoParentSpan",
		StartTime:   time.Now(),
		EndTime:     time.Now(),
		SpanContext: spanCtx,
		Attributes: []attribute.KeyValue{
			attribute.String("event", "{\"key1\":\"value1\",\"key2\":\"value2\",\"key3\":\"value3\"}"),
			attribute.String("response", "Hello"),
		},
		Resource: resource.NewWithAttributes(semconv.SchemaURL,
			attribute.KeyValue{
				Key:   "cloud.region",
				Value: attribute.StringValue("us-east-1"),
			},
			attribute.KeyValue{
				Key:   "lumigo_token",
				Value: attribute.StringValue("test"),
			},
		),
	}

	exp, err := newExporter(false)
	assert.NoError(e.T(), err)

	err = exp.ExportSpans(context.Background(), []trace.ReadOnlySpan{
		startSpan.Snapshot(),
		endSpan.Snapshot(),
	})
	assert.NoError(e.T(), err)

	startSpans, err := readSpansFromFile(SPAN_START_FILE)
	assert.NoError(e.T(), err)

	lumigoStart := startSpans[0]
	containerID := fmt.Sprint(startSpan.Attributes[0].Value.AsString())
	accountID := fmt.Sprint(startSpan.Attributes[1].Value.AsString())
	assert.Equal(e.T(), endSpan.SpanContext.SpanID().String(), lumigoStart.ID)
	assert.Equal(e.T(), endSpan.SpanContext.TraceID().String(), lumigoStart.TransactionID)
	assert.Equal(e.T(), containerID, lumigoStart.LambdaContainerID)
	assert.Equal(e.T(), accountID, lumigoStart.Account)

	endSpans, err := readSpansFromFile(SPAN_END_FILE)
	assert.NoError(e.T(), err)

	lumigoEnd := endSpans[0]
	event := fmt.Sprint(endSpan.Attributes[0].Value.AsString())
	response := fmt.Sprint(endSpan.Attributes[1].Value.AsString())
	assert.Equal(e.T(), endSpan.SpanContext.SpanID().String(), lumigoEnd.ID)
	assert.Equal(e.T(), endSpan.SpanContext.TraceID().String(), lumigoEnd.TransactionID)
	assert.Equal(e.T(), event, lumigoEnd.Event)
	assert.Equal(e.T(), response, lumigoEnd.LambdaResponse)
	assert.Equal(e.T(), endSpan.Resource.Attributes()[0].Value.AsString(), lumigoEnd.Region)
	assert.Equal(e.T(), endSpan.Resource.Attributes()[1].Value.AsString(), lumigoEnd.Token)

}

func readSpansFromFile(filePath string) ([]telemetry.Span, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return []telemetry.Span{}, err
	}
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return []telemetry.Span{}, err
	}
	var spans []telemetry.Span
	err = json.Unmarshal(byteValue, &spans)
	if err != nil {
		return []telemetry.Span{}, err
	}
	return spans, nil
}
