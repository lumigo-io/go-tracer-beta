package lumingo

import (
	"context"
	"encoding/json"
	"os"
	"reflect"

	log "github.com/sirupsen/logrus"
)

type contextKey string

const (
	wrappedByLumuingoKey contextKey = "isWrapped"
)

var logger *log.Logger

func init() {
	logger = log.New()
	logger.Out = os.Stdout
	logger.Formatter = &log.JSONFormatter{}
}

// WrapHandler wraps the lambda handler with lumingo tracer
func WrapHandler(handler interface{}, cfg *Config) interface{} {
	return func(ctx context.Context, msg json.RawMessage) (interface{}, error) {
		// if it's test do not recover raise errors
		if !cfg.isTest {
			defer func() {
				if err := recover(); err != nil { //nolint
					//probably log every error
				}
			}()
		}
		if !cfg.Enabled {
			return invoke(ctx, msg, handler)
		}
		_, ok := ctx.Value(wrappedByLumuingoKey).(bool)
		if !ok {
			ctx = context.WithValue(ctx, wrappedByLumuingoKey, true)
		}
		return invoke(ctx, msg, handler)
	}
}

// invoke will start invoking the lambda handler
func invoke(ctx context.Context, msg json.RawMessage, handler interface{}) (interface{}, error) {

	ev, err := unmarshalEventForHandler(msg, handler)
	if err != nil {
		return nil, err
	}

	handlerType := reflect.TypeOf(handler)
	args := []reflect.Value{}

	// detects the format of the handler function if includes
	// a. only context
	// b. only event
	// c. includes both context and event
	if handlerType.NumIn() == 1 {
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
		firstArgType := handlerType.In(0)
		if firstArgType.Implements(contextType) {
			args = []reflect.Value{reflect.ValueOf(ctx)}
		} else {
			args = []reflect.Value{ev.Elem()}
		}
	} else if handlerType.NumIn() == 2 {
		args = []reflect.Value{reflect.ValueOf(ctx), ev.Elem()}
	}

	handlerValue := reflect.ValueOf(handler)
	output := handlerValue.Call(args)

	var response interface{}
	var errResponse error

	// detect error response
	if len(output) > 0 {
		val := output[len(output)-1].Interface()
		if errVal, ok := val.(error); ok {
			errResponse = errVal
		}
	}

	// detect the real response
	if len(output) > 1 {
		response = output[0].Interface()
	}

	return response, errResponse
}

// unmarshalEventForHandler
func unmarshalEventForHandler(ev json.RawMessage, handler interface{}) (reflect.Value, error) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.NumIn() == 0 {
		return reflect.ValueOf(nil), nil
	}

	messageType := handlerType.In(handlerType.NumIn() - 1)
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	firstArgType := handlerType.In(0)

	if handlerType.NumIn() == 1 && firstArgType.Implements(contextType) {
		return reflect.ValueOf(nil), nil
	}

	newMessage := reflect.New(messageType)
	err := json.Unmarshal(ev, newMessage.Interface())
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	return newMessage, err
}
