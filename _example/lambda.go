package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	lumigotracer "github.com/lumigo-io/go-tracer-beta"
)

type MyEvent struct {
	Name string `json:"name"`
}

func HandleRequest(ctx context.Context, name MyEvent) (events.APIGatewayProxyResponse, error) {
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
		Token: "t_f2956385a53a4dcb9aea0",
	})
	lambda.Start(wrappedHandler)
}
