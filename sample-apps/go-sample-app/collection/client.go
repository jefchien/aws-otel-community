package collection

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var cfg = GetConfiguration()

const serviceName = "go"

var testingId = ""

var traceCommonLabels = []attribute.KeyValue{
	attribute.String("signal", "trace"),
	attribute.String("language", serviceName),
	attribute.String("host", cfg.Host),
	attribute.String("port", cfg.Port),
}

// StartClient starts the traces and metrics providers which periodically collects signals and exports them.
// Trace exporter and Metric exporter are both configured.
func StartClient(ctx context.Context) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if id, present := os.LookupEnv("INSTANCE_ID"); present {
		testingId = "_" + id
	}
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("otel-sample-app"),
	)
	if _, present := os.LookupEnv("OTEL_RESOURCE_ATTRIBUTES"); present {
		envResource, err := resource.New(ctx, resource.WithFromEnv())
		if err != nil {
			return nil, nil, err
		}
		res = envResource
	}

	// Setup trace related
	tp, err := setupTraceProvider(ctx, res)
	if err != nil {
		return tp, nil, err
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{}) // Set AWS X-Ray propagator

	return tp, func(context.Context) (err error) {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		err = tp.Shutdown(ctx)
		if err != nil {
			return err
		}
		return nil
	}, nil
}

// setupTraceProvider configures a trace exporter and an AWS X-Ray ID Generator.
func setupTraceProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	// INSECURE !! NOT TO BE USED FOR ANYTHING IN PRODUCTION
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())

	if err != nil {
		return nil, err
	}

	idg := xray.NewIDGenerator()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithIDGenerator(idg),
	)
	return tp, nil
}
