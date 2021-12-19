package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws/arn"
	lumigoctx "github.com/lumigo-io/go-tracer/internal/context"
	"github.com/lumigo-io/go-tracer/internal/telemetry"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	apitrace "go.opentelemetry.io/otel/trace"
)

func Span(ctx context.Context, span sdktrace.ReadOnlySpan, logger logrus.FieldLogger) telemetry.Span {
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
		TransactionID:    ksuid.New().String(),
		ParentID:         parentSpanID,
		StartedTimestamp: span.StartTime().Unix(),
		EndedTimestamp:   span.EndTime().Unix(),
	}

	lambdaCtx, lambdaOk := lambdacontext.FromContext(ctx)
	if lambdaOk {
		lumigoSpan.LambdaContainerID = lambdaCtx.AwsRequestID

		accountID, err := getAccountID(lambdaCtx)
		if err != nil {
			logger.WithError(err).Error()
		}
		lumigoSpan.Account = accountID

		deadline, _ := ctx.Deadline()
		lumigoSpan.MaxFinishTime = time.Now().Unix() - deadline.Unix()
	} else {
		logger.Error("unable to fetch from LambdaContext")
	}

	if token, ok := attrs["lumigo_token"]; ok {
		lumigoSpan.Token = fmt.Sprint(token)
	}

	if event, ok := attrs["event"]; ok {
		lumigoSpan.Event = fmt.Sprint(event)
	}

	if returnValue, ok := attrs["response"]; ok {
		lumigoSpan.LambdaResponse = fmt.Sprint(returnValue)
	}

	lumigoSpan.Region = os.Getenv("AWS_REGION")
	lumigoSpan.MemoryAllocated = os.Getenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE")
	lumigoSpan.Runtime = os.Getenv("AWS_EXECUTION_ENV")
	lumigoSpan.LambdaName = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	lumigoSpan.SpanInfo = telemetry.SpanInfo{
		LogStreamName: os.Getenv("AWS_LAMBDA_LOG_STREAM_NAME"),
		LogGroupName:  os.Getenv("AWS_LAMBDA_LOG_GROUP_NAME"),
		TraceID: telemetry.SpanTraceRoot{
			Root: getAmazonTraceID(),
		},
	}

	lumigoCtx, lumigoOk := lumigoctx.FromContext(ctx)
	if lumigoOk {
		lumigoSpan.SpanInfo.TracerVersion = telemetry.TracerVersion{
			Version: lumigoCtx.TracerVersion,
		}
	} else {
		logger.Error("unable to fetch from LumigoContext")
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

func getAccountID(ctx *lambdacontext.LambdaContext) (string, error) {
	functionARN, err := arn.Parse(ctx.InvokedFunctionArn)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse ARN")
	}
	return functionARN.AccountID, nil
}

func getAmazonTraceID() string {
	awsTraceItems := strings.SplitN(os.Getenv("_X_AMZN_TRACE_ID"), ";", 2)
	if len(awsTraceItems) > 1 {
		root := strings.SplitN(awsTraceItems[0], "=", 2)
		return root[1]
	}
	return ""
}
