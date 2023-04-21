package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/aws-otel-commnunity/sample-apps/go-sample-app/collection"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	traceXray = "xray"
	traceOtlp = "otlp"

	sampleAwsSDK  = "aws"
	testHttpCalls = "http"
)

var (
	traceOptions = []string{traceXray, traceOtlp}
	testOptions  = []string{sampleAwsSDK, testHttpCalls}
)

type response struct {
	TraceID string `json:"traceId"`
}

// This sample application is in conformance with the ADOT SampleApp requirements spec.
func main() {
	var traceType, testType string
	flag.StringVar(&traceType, "trace", traceXray, fmt.Sprintf("The type of trace data sent. Can be %v", traceOptions))
	flag.StringVar(&testType, "sample", sampleAwsSDK, fmt.Sprintf("The type of sample to run. Can be %v", testOptions))
	flag.Parse()

	log.Printf("Running sample app for %s/%s", traceType, testType)

	ctx := context.Background()

	// The seed for 'random' values used in this application
	rand.Seed(time.Now().UnixNano())

	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	var sampler collection.TraceSampler
	switch traceType {
	case traceXray:
		sampler = collection.XraySampler{}
	case traceOtlp:
		// Client starts
		tp, shutdown, err := collection.StartClient(ctx)
		if err != nil {
			log.Fatal(err)
		}
		defer shutdown(ctx)

		sampler = collection.OtlpSampler{TraceProvider: tp}
	default:
		log.Fatalf("invalid trace type: %s. see -h for options", traceType)
	}

	var traceID string
	switch testType {
	case sampleAwsSDK:
		traceID = sampler.AwsSdkCall(ctx)
	case testHttpCalls:
		traceID = sampler.OutgoingHttpCall(ctx, client)
	default:
		log.Fatalf("invalid test type: %s. see -h for options", testType)
	}
	if ot, ok := sampler.(collection.OtlpSampler); ok {
		_ = ot.TraceProvider.ForceFlush(ctx)
	}
	log.Printf("X-Ray ID: %s", traceID)

	//// Client starts
	//shutdown, err := collection.StartClient(ctx)
	//if err != nil {
	//    log.Fatal(err)
	//}
	//defer shutdown(ctx)
	//
	//cfg := collection.GetConfiguration()
	//
	//s3Client, err := collection.NewS3Wrapper()
	//if err != nil {
	//    fmt.Println(err)
	//}
	//// Creates a router, client and web server with several endpoints
	//r := mux.NewRouter()
	//client := http.Client{
	//    Transport: otelhttp.NewTransport(http.DefaultTransport),
	//}
	//
	//r.Use(otelmux.Middleware("sample-server"))
	//
	//// Three endpoints
	//r.HandleFunc("/otlp-aws-sdk-call", func(w http.ResponseWriter, r *http.Request) {
	//    traceID := collection.AwsSdkCall(r.Context(), s3Client)
	//    writeResponse(traceID, w)
	//})
	//
	//r.HandleFunc("/otlp-outgoing-http-call", func(w http.ResponseWriter, r *http.Request) {
	//    traceID := collection.OutgoingHttpCall(r.Context(), client)
	//    writeResponse(traceID, w)
	//})
	//
	//r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//    w.WriteHeader(http.StatusOK)
	//})
	//
	//// Root endpoint
	//http.Handle("/", r)
	//
	//srv := &http.Server{
	//    Addr: net.JoinHostPort(cfg.Host, cfg.Port),
	//}
	//fmt.Println("Listening on port:", srv.Addr)
	//log.Fatal(srv.ListenAndServe())
}

func writeResponse(traceID string, w http.ResponseWriter) {
	payload, _ := json.Marshal(response{TraceID: traceID})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}
