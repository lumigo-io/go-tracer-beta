package lumigotracer

import (
	"context"
	"io"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

type Transport struct {
	http.RoundTripper
}

func NewTransport(transport http.RoundTripper) *Transport {
	return &Transport{
		RoundTripper: transport,
	}
}

func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	traceCtx, span := otel.GetTracerProvider().Tracer("lumigo").Start(req.Context(), "HttpSpan")

	req = req.WithContext(traceCtx)
	span.SetAttributes(semconv.HTTPClientAttributesFromHTTPRequest(req)...)

	if req.Body != nil {
		buf := new(strings.Builder)
		body, bodyErr := io.Copy(buf, req.Body)
		if bodyErr != nil {
			return t.RoundTripper.RoundTrip(req)
		}
		span.SetAttributes(attribute.String("http.request_body", string(body)))
	}

	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		return
	}

	// response
	span.SetAttributes(semconv.HTTPAttributesFromHTTPStatusCode(resp.StatusCode)...)
	span.SetStatus(semconv.SpanStatusFromHTTPStatusCode(resp.StatusCode))
	if resp.Body != nil {
		buf := new(strings.Builder)
		body, bodyErr := io.Copy(buf, resp.Body)
		if bodyErr != nil {
			return
		}
		span.SetAttributes(attribute.String("http.response_body", string(body)))
	}
	resp.Body = &wrappedBody{ctx: traceCtx, span: span, body: resp.Body}

	return resp, err
}

type wrappedBody struct {
	ctx  context.Context
	span trace.Span
	body io.ReadCloser
}

var _ io.ReadCloser = &wrappedBody{}

func (wb *wrappedBody) Read(b []byte) (int, error) {
	n, err := wb.body.Read(b)

	switch err {
	case nil:
		// nothing to do here but fall through to the return
	case io.EOF:
		wb.span.End()
	default:
		wb.span.RecordError(err)
		wb.span.SetStatus(codes.Error, err.Error())
	}
	return n, err
}

func (wb *wrappedBody) Close() error {
	wb.span.End()
	return wb.body.Close()
}
