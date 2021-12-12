package transform

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/lumigo-io/go-tracer/internal/telemetry"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	apitrace "go.opentelemetry.io/otel/trace"
)

func Span(span sdktrace.ReadOnlySpan) telemetry.Span {
	numAttrs := len(span.Attributes()) + span.Resource().Len() + 2

	// If kind has been set, make room for it.
	if span.SpanKind() != apitrace.SpanKindUnspecified {
		numAttrs++
	}

	// Status of Ok and Unset are not considered errors.
	isError := span.Status().Code == codes.Error
	if isError {
		numAttrs += 2
	}

	attrs := make(map[string]interface{}, numAttrs)

	for iter := span.Resource().Iter(); iter.Next(); {
		kv := iter.Label()
		attrs[string(kv.Key)] = kv.Value.AsInterface()
	}
	for _, kv := range span.Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsInterface()
	}

	if span.SpanKind() != apitrace.SpanKindUnspecified {
		attrs["span.kind"] = strings.ToLower(span.SpanKind().String())
	}

	parentSpanID := ""
	if span.Parent().IsValid() {
		parentSpanID = span.Parent().SpanID().String()
	}
	lumigoSpan := telemetry.Span{
		ID:               span.SpanContext().SpanID().String(),
		TransactionID:    span.SpanContext().TraceID().String(),
		ParentID:         parentSpanID,
		StartedTimestamp: span.StartTime(),
		EndedTimestamp:   span.EndTime(),
	}

	if id, ok := attrs["faas.execution"]; ok {
		lumigoSpan.LambdaContainerID = fmt.Sprint(id)
	}

	if region, ok := attrs["cloud.region"]; ok {
		lumigoSpan.Region = fmt.Sprint(region)
	}

	if token, ok := attrs["lumigo_token"]; ok {
		lumigoSpan.Token = fmt.Sprint(token)
	}

	if accountID, ok := attrs["cloud.account.id"]; ok {
		lumigoSpan.Account = fmt.Sprint(accountID)
	}

	if event, ok := attrs["event"]; ok {
		lumigoSpan.Event = fmt.Sprint(event)
	}

	if returnValue, ok := attrs["response"]; ok {
		lumigoSpan.LambdaResponse = fmt.Sprint(returnValue)
	}

	lumigoSpan.MemoryAllocated = os.Getenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE")
	lumigoSpan.Runtime = os.Getenv("AWS_EXECUTION_ENV")
	lumigoSpan.LambdaName = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	lumigoSpan.SpanInfo = telemetry.SpanInfo{
		LogStreamName: os.Getenv("AWS_LAMBDA_LOG_STREAM_NAME"),
		LogGroupName:  os.Getenv("AWS_LAMBDA_LOG_GROUP_NAME"),
		TraceID:       telemetry.SpanTraceRoot{},
	}
	awsTraceID := strings.SplitN(os.Getenv("_X_AMZN_TRACE_ID"), "=", 2)
	if len(awsTraceID) > 2 {
		lumigoSpan.SpanInfo.TraceID.Root = awsTraceID[1]
	}
	isColdStart := os.Getenv("IS_COLD_START")
	if isColdStart == "" && !isProvisionConcurrencyInitialization() {
		lumigoSpan.LambdaReadiness = "cold"
	} else {
		lumigoSpan.LambdaReadiness = "warm"
	}

	lambdaType := "function"
	if span.Name() != lumigoSpan.LambdaName && span.Name() != "LumigoParentSpan" {
		lambdaType = "http"
	}
	lumigoSpan.LambdaType = lambdaType

	envs := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		envs[pair[0]] = pair[1]
	}
	envsString, _ := json.Marshal(envs)
	lumigoSpan.LambdaEnvVars = string(envsString)
	return lumigoSpan
}

func isProvisionConcurrencyInitialization() bool {
	return os.Getenv("AWS_LAMBDA_INITIALIZATION_TYPE") == "provisioned-concurrency"
}
