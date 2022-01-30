package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	lumigotracer "github.com/lumigo-io/go-tracer-beta"
)

type MyEvent struct {
	Name string `json:"name"`
}

func HandleRequest(ctx context.Context, name MyEvent) (string, error) {
	response := fmt.Sprintf("Hello %s!", name.Name)
	returnErr, ok := os.LookupEnv("RETURN_ERROR")
	if !ok {
		return response, nil
	}
	isReturnErr, err := strconv.ParseBool(returnErr)
	if err != nil {
		return response, nil
	}
	if isReturnErr {
		return "", errors.New("failed error")
	}
	return response, nil
}

func main() {
	os.Setenv("LUMIGO_DEBUG", "true")
	wrappedHandler := lumigotracer.WrapHandler(HandleRequest, &lumigotracer.Config{
		PrintStdout: false,
		Token:       "t_f2956385a53a4dcb9aea0",
	})
	lambda.Start(wrappedHandler)
}
