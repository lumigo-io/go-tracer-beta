package lumigotracer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/stretchr/testify/assert"
)

var (
	mockLambdaContext = lambdacontext.LambdaContext{
		AwsRequestID:       "123",
		InvokedFunctionArn: "arn:partition:service:region:account-id:resource-type:resource-id",
		Identity: lambdacontext.CognitoIdentity{
			CognitoIdentityID:     "someId",
			CognitoIdentityPoolID: "somePoolId",
		},
		ClientContext: lambdacontext.ClientContext{},
	}
	mockContext = lambdacontext.NewContext(context.TODO(), &mockLambdaContext)
)

type expected struct {
	val interface{}
	err error
}

func TestLambdaHandlerSignatures(t *testing.T) {
	logger.Out = io.Discard
	hello := func(s string) string {
		return fmt.Sprintf("Hello %s!", s)
	}

	testCases := []struct {
		name     string
		input    interface{}
		expected expected
		handler  interface{}
	}{
		{
			name:     "input: string, no context",
			input:    "test",
			expected: expected{`"Hello test!"`, nil},
			handler: func(name string) (string, error) {
				return hello(name), nil
			},
		},
		{
			name:     "input: string, with context",
			input:    "test",
			expected: expected{`"Hello test!"`, nil},
			handler: func(ctx context.Context, name string) (string, error) {
				return hello(name), nil
			},
		},
		{
			name:     "input: none, error on return",
			input:    nil,
			expected: expected{"", errors.New("failed")},
			handler: func() (interface{}, error) {
				return nil, errors.New("failed")
			},
		},
		{
			name:     "input: event, error on return",
			input:    "test",
			expected: expected{"", errors.New("failed")},
			handler: func(e interface{}) (interface{}, error) {
				return nil, errors.New("failed")
			},
		},
		{
			name:     "input: context & event, error on return",
			input:    "test",
			expected: expected{"", errors.New("failed")},
			handler: func(ctx context.Context, e interface{}) (interface{}, error) {
				return nil, errors.New("failed")
			},
		},
		{
			name:     "input: event, lambda Invoke error on return",
			input:    "test",
			expected: expected{"", messages.InvokeResponse_Error{Message: "message", Type: "type"}},
			handler: func(e interface{}) (interface{}, error) {
				return nil, messages.InvokeResponse_Error{Message: "message", Type: "type"}
			},
		},
		{
			name:     "input: struct event, response number",
			input:    struct{ Port int }{9090},
			expected: expected{`9090`, nil},
			handler: func(event struct{ Port int }) (int, error) {
				return event.Port, nil
			},
		},
		{
			name:     "input: struct event, response as struct",
			input:    9090,
			expected: expected{`{"Port":9090}`, nil},
			handler: func(event int) (struct{ Port int }, error) {
				return struct{ Port int }{event}, nil
			},
		},
	}
	// test invocation via a Handler
	for i, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("handlerTestCase[%d] %s", i, testCase.name), func(t *testing.T) {
			inputPayload, _ := json.Marshal(testCase.input)

			lambdaHandler := WrapHandler(testCase.handler, &Config{Token: "token"})
			handler := reflect.ValueOf(lambdaHandler)
			handlerType := handler.Type()

			response := handler.Call([]reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(inputPayload)})

			if testCase.expected.err != nil {
				assert.Equal(t, testCase.expected.err, response[handlerType.NumOut()-1].Interface())
			} else {
				assert.Nil(t, response[handlerType.NumOut()-1].Interface())
				responseValMarshalled, _ := json.Marshal(response[0].Interface())
				assert.Equal(t, testCase.expected.val, string(responseValMarshalled))
			}
		})
	}
}
