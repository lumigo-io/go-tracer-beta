package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/google/uuid"
	lumigoctx "github.com/lumigo-io/go-tracer-beta/internal/context"
	"github.com/lumigo-io/go-tracer-beta/internal/telemetry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	apitrace "go.opentelemetry.io/otel/trace"
)

type mapper struct {
	ctx    context.Context
	span   sdktrace.ReadOnlySpan
	logger logrus.FieldLogger
}

func NewMapper(ctx context.Context, span sdktrace.ReadOnlySpan, logger logrus.FieldLogger) *mapper {
	return &mapper{
		ctx:    ctx,
		span:   span,
		logger: logger,
	}
}

func (m *mapper) Transform() telemetry.Span {
	numAttrs := len(m.span.Attributes()) + m.span.Resource().Len() + 2

	if m.span.SpanKind() != apitrace.SpanKindUnspecified {
		numAttrs++
	}

	attrs := make(map[string]interface{}, numAttrs)

	for iter := m.span.Resource().Iter(); iter.Next(); {
		kv := iter.Label()
		attrs[string(kv.Key)] = kv.Value.AsInterface()
	}
	for _, kv := range m.span.Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsInterface()
	}

	m.logger.WithFields(attrs).Info("span attributes")

	if m.span.SpanKind() != apitrace.SpanKindUnspecified {
		attrs["m.span.kind"] = strings.ToLower(m.span.SpanKind().String())
	}

	lumigoSpan := telemetry.Span{
		StartedTimestamp: m.span.StartTime().UnixMilli(),
		EndedTimestamp:   m.span.EndTime().UnixMilli(),
	}

	isStartSpan := telemetry.IsStartSpan(m.span)
	lambdaCtx, lambdaOk := lambdacontext.FromContext(m.ctx)
	if lambdaOk {
		uuid, _ := uuid.NewUUID()
		lumigoSpan.LambdaContainerID = uuid.String()
		lumigoSpan.ID = lambdaCtx.AwsRequestID

		if isStartSpan {
			lumigoSpan.ID = fmt.Sprintf("%s_started", lumigoSpan.ID)
		}

		accountID, err := getAccountID(lambdaCtx)
		if err != nil {
			m.logger.WithError(err).Error()
		}
		lumigoSpan.Account = accountID

		deadline, _ := m.ctx.Deadline()
		lumigoSpan.MaxFinishTime = time.Now().UnixMilli() - deadline.UnixMilli()
	} else {
		m.logger.Error("unable to fetch from LambdaContext")
	}

	if token, ok := attrs["lumigo_token"]; ok {
		lumigoSpan.Token = fmt.Sprint(token)
	} else {
		m.logger.Error("unable to fetch lumigo token from span")
	}

	if event, ok := attrs["event"]; ok {
		lumigoSpan.Event = fmt.Sprint(event)
	} else {
		m.logger.Error("unable to fetch lambda event from span")
	}

	if returnValue, ok := attrs["response"]; ok {
		lumigoSpan.LambdaResponse = aws.String(fmt.Sprint(returnValue))
	} else if !isStartSpan {
		m.logger.Error("unable to fetch lambda response from span")
	}

	lumigoSpan.Region = os.Getenv("AWS_REGION")
	lumigoSpan.MemoryAllocated = os.Getenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE")
	lumigoSpan.Runtime = os.Getenv("AWS_EXECUTION_ENV")
	lumigoSpan.LambdaName = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	awsRoot := getAmazonTraceID()
	if awsRoot == "" {
		m.logger.Error("unable to fetch Amazon Trace ID")
	}
	lumigoSpan.SpanInfo = telemetry.SpanInfo{
		LogStreamName: os.Getenv("AWS_LAMBDA_LOG_STREAM_NAME"),
		LogGroupName:  os.Getenv("AWS_LAMBDA_LOG_GROUP_NAME"),
		TraceID: telemetry.SpanTraceRoot{
			Root: awsRoot,
		},
	}

	if transactionID := getTransactionID(awsRoot); transactionID != "" {
		lumigoSpan.TransactionID = transactionID
	} else {
		m.logger.Error("unable to fetch transaction ID")
	}

	lumigoCtx, lumigoOk := lumigoctx.FromContext(m.ctx)
	if lumigoOk {
		lumigoSpan.SpanInfo.TracerVersion = telemetry.TracerVersion{
			Version: lumigoCtx.TracerVersion,
		}
	} else {
		m.logger.Error("unable to fetch from LumigoContext")
	}

	isWarmStart := os.Getenv("IS_WARM_START")
	if isWarmStart == "" && !isProvisionConcurrencyInitialization() {
		lumigoSpan.LambdaReadiness = "cold"
	} else {
		lumigoSpan.LambdaReadiness = "warm"
	}

	lambdaType := "function"
	if m.span.Name() != lumigoSpan.LambdaName && m.span.Name() != "LumigoParentSpan" {
		lambdaType = "http"
	}
	lumigoSpan.LambdaType = lambdaType

	if !isStartSpan {
		lumigoSpan.SpanError = m.getSpanError(attrs)
	}
	lumigoSpan.LambdaEnvVars = m.getEnvVars()
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

func getTransactionID(root string) string {
	items := strings.SplitN(root, "-", 3)
	if len(items) > 1 {
		return items[2]
	}
	return ""
}

func (m *mapper) getSpanError(attrs map[string]interface{}) *telemetry.SpanError {
	if _, ok := attrs["has_error"]; !ok {
		return nil
	}
	var spanError telemetry.SpanError
	if errType, ok := attrs["error_type"]; ok {
		spanError.Type = fmt.Sprint(errType)
	} else {
		m.logger.Error("unable to fetch lambda error type from span")
	}

	if errMessage, ok := attrs["error_message"]; ok {
		spanError.Message = fmt.Sprint(errMessage)
	} else {
		m.logger.Error("unable to fetch lambda error message from span")
	}

	if errStacktrace, ok := attrs["error_stacktrace"]; ok {
		spanError.Stacktrace = fmt.Sprint(errStacktrace)
	} else {
		m.logger.Error("unable to fetch lambda error stacktrace from span")
	}
	if spanError.IsEmpty() {
		return nil
	}
	return &spanError
}

func (m *mapper) getEnvVars() string {
	envs := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		envs[pair[0]] = pair[1]
	}
	envsString, err := json.Marshal(envs)
	if err != nil {
		m.logger.Error("unable to fetch lambda environment vars")
	}
	return string(envsString)
}
