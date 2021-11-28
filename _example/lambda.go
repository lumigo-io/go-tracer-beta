package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	lumigotracer "github.com/lumigo-io/go-tracer"
)

type MyEvent struct {
	Name string `json:"name"`
}

var client *http.Client
var awsConfig aws.Config

func init() {
	client = &http.Client{
		Transport: lumigotracer.NewTransport(http.DefaultTransport),
	}
	awsConfig, _ = config.LoadDefaultConfig(context.Background())
}

func HandleRequest(ctx context.Context, name MyEvent) (string, error) {
	// # Solution 1
	// OpenTelemetry HTTP tracking for AWS
	// cfg, err := awsConfig.LoadDefaultConfig(ctx, config.WithHTTPClient(client))

	// # Solution 2
	// OpenTelemetry HTTP tracking for AWS
	// cfg := aws.Config{
	// 	Region:     "us-east-1",
	// 	HTTPClient: client,
	// }

	// # Solution3
	// OpenTelemetry AWS Clients traffic middleware
	// cfg := aws.Config{
	// 	Region:	"us-east-1",
	//}
	// lumigotracer.TraceAWSClients(&cfg)

	// # Solution 4
	// OpenTelemetry AWS Clients traffic middleware
	// cfg, err := lumigotracer.LoadAWSConfig(ctx)
	// if err != nil {
	// 	return "", err
	// }
	s3Client := s3.NewFromConfig(awsConfig)
	input := &s3.ListBucketsInput{}
	result, err := s3Client.ListBuckets(ctx, input)
	if err != nil {
		return "", err
	}

	// track external requests
	req, err := http.NewRequest("GET", "http://google.com", nil)
	client.Do(req)
	for _, bucket := range result.Buckets {
		log.Println(*bucket.Name + ": " + bucket.CreationDate.Format("2006-01-02 15:04:05 Monday"))
	}
	return fmt.Sprintf("Hello %s!", name.Name), nil
}

func main() {
	wrappedHandler := lumigotracer.WrapHandlerWithAWSConfig(HandleRequest, &lumigotracer.Config{
		PrintStdout: true,
		Token:       "token",
	}, &awsConfig)

	lambda.Start(wrappedHandler)
}
