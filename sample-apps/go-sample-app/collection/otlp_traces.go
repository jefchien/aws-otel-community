package collection

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Contains all of the endpoint logic.

type TraceSampler interface {
	AwsSdkCall(ctx context.Context) string
	OutgoingHttpCall(ctx context.Context, client http.Client) string
}

type OtlpSampler struct {
	TraceProvider *sdktrace.TracerProvider
}

// AwsSdkCall mocks a request to s3. ListBuckets are nil so no credentials are needed.
// Generates an Xray Trace ID.
func (t OtlpSampler) AwsSdkCall(ctx context.Context) string {
	tracer := otel.Tracer("demo")
	spanCtx, span := tracer.Start(ctx, "otel-sample-app")
	defer span.End()

	innerSpanCtx, innerSpan := tracer.Start(
		spanCtx,
		"aws-sdk-call",
		trace.WithAttributes(traceCommonLabels...),
	)
	defer innerSpan.End()

	awscfg, err := config.LoadDefaultConfig(innerSpanCtx)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	otelaws.AppendMiddlewares(&awscfg.APIOptions)

	s3Client := s3.NewFromConfig(awscfg)
	_, _ = s3Client.ListBuckets(innerSpanCtx, &s3.ListBucketsInput{}) // nil or else would need real aws credentials

	return getXrayTraceID(span)
}

// OutgoingHttpCall makes an HTTP GET request to https://aws.amazon.com/ and generates an Xray Trace ID.
func (t OtlpSampler) OutgoingHttpCall(ctx context.Context, client http.Client) string {
	tracer := otel.Tracer("demo")
	spanCtx, span := tracer.Start(ctx, "otel-sample-app")
	defer span.End()

	innerCtx, innerSpan := tracer.Start(
		spanCtx,
		"outgoing-http-call",
		trace.WithAttributes(traceCommonLabels...),
	)
	defer innerSpan.End()

	req, _ := http.NewRequestWithContext(innerCtx, "GET", "https://aws.amazon.com/", nil)
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}

	defer res.Body.Close()

	return getXrayTraceID(span)
}

// getXrayTraceID generates a trace ID in Xray format from the span context.
func getXrayTraceID(span trace.Span) string {
	xrayTraceID := span.SpanContext().TraceID().String()
	return fmt.Sprintf("1-%s-%s", xrayTraceID[0:8], xrayTraceID[8:])
}
