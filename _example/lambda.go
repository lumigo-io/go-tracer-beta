package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	lumigotracer "github.com/lumigo-io/go-tracer-beta"
)

var client *http.Client

type MyEvent struct {
	Name string `json:"name"`
}

func init() {
	client = &http.Client{
		Transport: lumigotracer.NewTransport(http.DefaultTransport),
	}
}

func HandleRequest(ctx context.Context, name MyEvent) (events.APIGatewayProxyResponse, error) {
	cfg, _ := lumigotracer.LoadAWSConfig(ctx)
	s3Client := s3.NewFromConfig(cfg)
	input := &s3.ListBucketsInput{}
	_, err := s3Client.ListBuckets(ctx, input)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: "", StatusCode: 500}, err
	}
	lumigotracer.TraceAWSClients(&cfg)
	response := fmt.Sprintf("Hello %s!", name.Name)
	returnErr, ok := os.LookupEnv("RETURN_ERROR")
	if !ok {
		return events.APIGatewayProxyResponse{Body: response, StatusCode: 200}, nil
	}
	isReturnErr, err := strconv.ParseBool(returnErr)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: response, StatusCode: 500}, err
	}
	if isReturnErr {
		return events.APIGatewayProxyResponse{Body: response, StatusCode: 500}, errors.New("failed error")
	}
	return events.APIGatewayProxyResponse{Body: response, StatusCode: 200}, nil
}

func main() {
	os.Setenv("LUMIGO_DEBUG", "true")
	wrappedHandler := lumigotracer.WrapHandler(HandleRequest, &lumigotracer.Config{
		Token:       "t_f2956385a53a4dcb9aea0",
		PrintStdout: true,
	})
	lambda.Start(wrappedHandler)
}
