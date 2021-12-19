package transform

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/lumigo-io/go-tracer/internal/telemetry"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var (
	traceID, _        = trace.TraceIDFromHex("000000")
	spanID, _         = trace.SpanIDFromHex("000000")
	mockLambdaContext = lambdacontext.LambdaContext{
		AwsRequestID:       "123",
		InvokedFunctionArn: "arn:partition:service:region:account-id:resource-type:resource-id",
		Identity: lambdacontext.CognitoIdentity{
			CognitoIdentityID:     "someId",
			CognitoIdentityPoolID: "somePoolId",
		},
	}
)

func TestTransform(t *testing.T) {
	now := time.Now()
	ctx := lambdacontext.NewContext(context.Background(), &mockLambdaContext)
	deadline, _ := ctx.Deadline()
	testcases := []struct {
		testname string
		input    *tracetest.SpanStub
		expect   telemetry.Span
		before   func()
		after    func()
	}{
		{
			testname: "simplest span",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: traceID,
					SpanID:  spanID,
				}),
				StartTime: now,
				EndTime:   now.Add(1 * time.Second),
				Name:      "test",
			},
			expect: telemetry.Span{
				LambdaName:        "test",
				LambdaType:        "function",
				LambdaReadiness:   "cold",
				LambdaContainerID: "123",
				Account:           "account-id",
				ID:                spanID.String(),
				StartedTimestamp:  now.Unix(),
				EndedTimestamp:    now.Add(1 * time.Second).Unix(),
				MaxFinishTime:     now.Unix() - deadline.Unix(),
			},
			before: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test")
			},
			after: func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
			},
		},
		{
			testname: "span with runtime and info",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: traceID,
					SpanID:  spanID,
				}),
				StartTime: now,
				EndTime:   now.Add(1 * time.Second),
				Name:      "test",
			},
			expect: telemetry.Span{
				LambdaName:        "test",
				LambdaType:        "function",
				LambdaReadiness:   "cold",
				LambdaContainerID: "123",
				Runtime:           "go",
				Account:           "account-id",
				SpanInfo: telemetry.SpanInfo{
					LogStreamName: "2021/12/06/[$LATEST]2f4f26a6224b421c86bc4570bb7bf84b",
					LogGroupName:  "/aws/lambda/helloworld-37",
					TraceID:       telemetry.SpanTraceRoot{},
				},
				ID:               spanID.String(),
				StartedTimestamp: now.Unix(),
				EndedTimestamp:   now.Add(1 * time.Second).Unix(),
				MaxFinishTime:    now.Unix() - deadline.Unix(),
			},
			before: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test")
				os.Setenv("AWS_EXECUTION_ENV", "go")
				os.Setenv("AWS_LAMBDA_LOG_STREAM_NAME", "2021/12/06/[$LATEST]2f4f26a6224b421c86bc4570bb7bf84b")
				os.Setenv("AWS_LAMBDA_LOG_GROUP_NAME", "/aws/lambda/helloworld-37")
			},
			after: func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
				os.Unsetenv("AWS_EXECUTION_ENV")
				os.Unsetenv("AWS_LAMBDA_LOG_STREAM_NAME")
				os.Unsetenv("AWS_LAMBDA_LOG_GROUP_NAME")
			},
		},
		{
			testname: "span lambda readiness warm",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: traceID,
					SpanID:  spanID,
				}),
				StartTime: now,
				EndTime:   now.Add(1 * time.Second),
				Name:      "test",
			},
			expect: telemetry.Span{
				LambdaName:        "test",
				LambdaType:        "function",
				LambdaReadiness:   "warm",
				LambdaContainerID: "123",
				Account:           "account-id",
				ID:                spanID.String(),
				StartedTimestamp:  now.Unix(),
				EndedTimestamp:    now.Add(1 * time.Second).Unix(),
				MaxFinishTime:     now.Unix() - deadline.Unix(),
			},
			before: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test")
				os.Setenv("IS_COLD_START", "true")
			},
			after: func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
				os.Unsetenv("IS_COLD_START")
			},
		},
		{
			testname: "span with event",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: traceID,
					SpanID:  spanID,
				}),
				StartTime: now,
				EndTime:   now.Add(1 * time.Second),
				Name:      "test",
				Attributes: []attribute.KeyValue{
					attribute.String("event", "test"),
				},
			},
			expect: telemetry.Span{
				LambdaName:        "test",
				LambdaType:        "function",
				LambdaReadiness:   "warm",
				LambdaContainerID: "123",
				Event:             "test",
				Account:           "account-id",
				ID:                spanID.String(),
				StartedTimestamp:  now.Unix(),
				EndedTimestamp:    now.Add(1 * time.Second).Unix(),
				MaxFinishTime:     now.Unix() - deadline.Unix(),
			},
			before: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test")
				os.Setenv("IS_COLD_START", "true")
			},
			after: func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
				os.Unsetenv("IS_COLD_START")
			},
		},
		{
			testname: "span with event and response",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: traceID,
					SpanID:  spanID,
				}),
				StartTime: now,
				EndTime:   now.Add(1 * time.Second),
				Name:      "test",
				Attributes: []attribute.KeyValue{
					attribute.String("event", "test"),
					attribute.String("response", "test2"),
				},
			},
			expect: telemetry.Span{
				LambdaName:        "test",
				LambdaType:        "function",
				LambdaReadiness:   "warm",
				LambdaResponse:    "test2",
				LambdaContainerID: "123",
				Event:             "test",
				Account:           "account-id",
				ID:                spanID.String(),
				StartedTimestamp:  now.Unix(),
				EndedTimestamp:    now.Add(1 * time.Second).Unix(),
				MaxFinishTime:     now.Unix() - deadline.Unix(),
			},
			before: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test")
				os.Setenv("IS_COLD_START", "true")
			},
			after: func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
				os.Unsetenv("IS_COLD_START")
			},
		},
		{
			testname: "span with LumigoParentSpan is function",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: traceID,
					SpanID:  spanID,
				}),
				StartTime: now,
				EndTime:   now.Add(1 * time.Second),
				Name:      "LumigoParentSpan",
				Attributes: []attribute.KeyValue{
					attribute.String("event", "test"),
					attribute.String("response", "test2"),
				},
			},
			expect: telemetry.Span{
				LambdaName:        "test",
				LambdaType:        "function",
				LambdaReadiness:   "warm",
				LambdaResponse:    "test2",
				LambdaContainerID: "123",
				Event:             "test",
				Account:           "account-id",
				ID:                spanID.String(),
				StartedTimestamp:  now.Unix(),
				EndedTimestamp:    now.Add(1 * time.Second).Unix(),
				MaxFinishTime:     now.Unix() - deadline.Unix(),
			},
			before: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test")
				os.Setenv("IS_COLD_START", "true")
			},
			after: func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
				os.Unsetenv("IS_COLD_START")
			},
		},
		{
			testname: "span from S3 or HTTP is type http",
			input: &tracetest.SpanStub{
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID: traceID,
					SpanID:  spanID,
				}),
				StartTime: now,
				EndTime:   now.Add(1 * time.Second),
				Name:      "S3 HTTP",
				Attributes: []attribute.KeyValue{
					attribute.String("event", "test"),
					attribute.String("response", "test2"),
				},
			},
			expect: telemetry.Span{
				LambdaName:        "test",
				LambdaType:        "http",
				LambdaReadiness:   "warm",
				LambdaResponse:    "test2",
				LambdaContainerID: "123",
				Event:             "test",
				Account:           "account-id",
				ID:                spanID.String(),
				StartedTimestamp:  now.Unix(),
				EndedTimestamp:    now.Add(1 * time.Second).Unix(),
				MaxFinishTime:     now.Unix() - deadline.Unix(),
			},
			before: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test")
				os.Setenv("IS_COLD_START", "true")
			},
			after: func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
				os.Unsetenv("IS_COLD_START")
			},
		},
	}

	for _, tc := range testcases {
		tc.before()
		lumigoSpan := Span(ctx, tc.input.Snapshot(), logrus.New())
		// intentionally ignore CI and Local envs
		lumigoSpan.LambdaEnvVars = ""
		// intentionally ignore generated transactionId
		lumigoSpan.TransactionID = ""
		if !reflect.DeepEqual(lumigoSpan, tc.expect) {
			t.Errorf("%s: %#v != %#v", tc.testname, lumigoSpan, tc.expect)
		}
		tc.after()
	}
}
