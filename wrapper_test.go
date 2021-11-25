package lumigotracer

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

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

func TestLambdaHandlerSignatures(t *testing.T) {
	emptyPayload := ""
	testCases := []struct {
		name     string
		handler  interface{}
		expected error
		args     []reflect.Value
	}{
		{
			name:     "nil handler",
			expected: errors.New("handler is nil"),
			handler:  nil,
			args:     []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "handler is a struct",
			expected: errors.New("handler kind struct is not func"),
			handler:  struct{}{},
			args:     []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "handler more than two args",
			expected: errors.New("handlers may not take more than two arguments, but handler takes 3"),
			handler: func(n context.Context, x string, y string) error {
				return nil
			},
			args: []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "handler first argument not a context",
			expected: errors.New("handler takes two arguments, but the first is not Context. got string"),
			handler: func(a string, x context.Context) error {
				return nil
			},
			args: []reflect.Value{reflect.ValueOf(mockContext), reflect.ValueOf(emptyPayload)},
		},
		{
			name:     "missing params & return no error",
			expected: nil,
			handler: func() {
			},
			args: []reflect.Value{reflect.ValueOf(mockContext)},
		},
	}
	for i, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("testCase[%d] %s", i, testCase.name), func(t *testing.T) {
			lambdaHandler := WrapHandler(testCase.handler, &Config{Token: "token"})
			handler := reflect.ValueOf(lambdaHandler)
			resp := handler.Call(testCase.args)
			assert.Equal(t, 2, len(resp))
			assert.Equal(t, testCase.expected, resp[1].Interface())
		})
	}
}
