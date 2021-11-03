package lumingo

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

type mockLambdaEvent struct {
	Event map[string]string `json:"event"`
	ID    string            `json:"id"`
}

func TestWrapperGatewayRequest(t *testing.T) {
	handler := func(ctx context.Context, request events.APIGatewayProxyRequest) (string, error) {
		assert.Equal(t, "c6af9ac6-7b61-11e6-9a41-93e8deadbeef", request.RequestContext.RequestID)
		return "test", nil
	}

	wrapped := WrapHandler(handler, &Config{isTest: true}).(func(context.Context, json.RawMessage) (interface{}, error))

	body := loadRawJSON(t, "../test/testdata/api-gt-proxy-event.json")
	response, err := wrapped(context.Background(), *body)
	assert.NoError(t, err)
	assert.Equal(t, "test", response)
}

func TestWrapperLambdaEvent(t *testing.T) {
	handler := func(ctx context.Context, message mockLambdaEvent) (string, error) {
		assert.Equal(t, "1234", message.ID)
		return "test", nil
	}

	wrapped := WrapHandler(handler, &Config{isTest: true}).(func(context.Context, json.RawMessage) (interface{}, error))
	body := loadRawJSON(t, "../test/testdata/lambda-event.json")
	response, err := wrapped(context.Background(), *body)
	assert.NoError(t, err)
	assert.Equal(t, "test", response)
}

func TestWrapperMissingContext(t *testing.T) {
	handler := func(message mockLambdaEvent) (string, error) {
		assert.Equal(t, "1234", message.ID)
		return "test", nil
	}

	wrapped := WrapHandler(handler, &Config{isTest: true}).(func(context.Context, json.RawMessage) (interface{}, error))
	body := loadRawJSON(t, "../test/testdata/lambda-event.json")
	response, err := wrapped(context.Background(), *body)
	assert.NoError(t, err)
	assert.Equal(t, "test", response)
}

func TestWrapperMissingEvent(t *testing.T) {
	handler := func(ctx context.Context) (string, error) {
		return "", nil
	}

	wrapped := WrapHandler(handler, &Config{isTest: true}).(func(context.Context, json.RawMessage) (interface{}, error))
	_, err := wrapped(context.Background(), nil)
	assert.NoError(t, err)
}

func TestWrapperMissingAllArgs(t *testing.T) {
	handler := func() (string, error) {
		return "", nil
	}

	wrapped := WrapHandler(handler, &Config{isTest: true}).(func(context.Context, json.RawMessage) (interface{}, error))
	_, err := wrapped(context.Background(), nil)
	assert.NoError(t, err)
}

func TestWrapperHandlerError(t *testing.T) {
	expectedErr := errors.New("failed")
	handler := func(message mockLambdaEvent) (string, error) {
		return "test", expectedErr
	}

	wrapped := WrapHandler(handler, &Config{isTest: true}).(func(context.Context, json.RawMessage) (interface{}, error))
	body := loadRawJSON(t, "../test/testdata/lambda-event.json")
	response, err := wrapped(context.Background(), *body)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, "test", response)
}

func TestWrapperHandlerErrorNoResponse(t *testing.T) {
	expectedErr := errors.New("failed")
	handler := func(message mockLambdaEvent) error {
		return expectedErr
	}

	wrapped := WrapHandler(handler, &Config{isTest: true}).(func(context.Context, json.RawMessage) (interface{}, error))
	body := loadRawJSON(t, "../test/testdata/lambda-event.json")
	response, err := wrapped(context.Background(), *body)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, nil, response)
}

func loadRawJSON(t *testing.T, filename string) *json.RawMessage {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		assert.Fail(t, "Couldn't find JSON file")
		return nil
	}
	msg := json.RawMessage{}
	msg.UnmarshalJSON(bytes) //nolint
	return &msg
}
