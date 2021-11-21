package lumigo

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

// NewTransport creates an HTTP transport
func NewTransport(transport http.RoundTripper) *otelhttp.Transport {
	return otelhttp.NewTransport(
		transport,
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
	)
}

// LoadAWSConfig returns an AWS config and we wrapped alredy the AWS
// clients with OpenTelemetry
func LoadAWSConfig(ctx context.Context, optFns ...func(*awsConfig.LoadOptions) error) (cfg aws.Config, err error) {
	cfg, err = awsConfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return aws.Config{}, err
	}
	otelaws.AppendMiddlewares(&cfg.APIOptions, otelaws.WithTracerProvider(otel.GetTracerProvider()))
	return cfg, err
}

// TraceAWSClients adds the middlewares for AWS Client
// with OpenTelemetry
func TraceAWSClients(cfg *aws.Config) {
	otelaws.AppendMiddlewares(&cfg.APIOptions, otelaws.WithTracerProvider(otel.GetTracerProvider()))
}
