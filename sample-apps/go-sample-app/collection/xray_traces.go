package collection

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-xray-sdk-go/xray"
)

type XraySampler struct {
}

func (t XraySampler) AwsSdkCall(ctx context.Context) string {
	// example of custom segment
	rootCtx, root := xray.BeginSegment(ctx, "xray-sample-app")
	defer root.Close(nil)

	if err := root.AddMetadata("description", "makes AWS SDK calls"); err != nil {
		log.Println(err)
	}

	subCtx, subSeg := xray.BeginSubsegment(rootCtx, "aws-sdk-calls")
	defer subSeg.Close(nil)

	if err := subSeg.AddMetadata("description", "makes an SQS and S3 call"); err != nil {
		log.Println(err)
	}
	if err := subSeg.AddMetadata("expected-results", map[string]interface{}{
		"SQS": 403,
		"S3":  200,
	}); err != nil {
		log.Println(err)
	}

	awsSess, err := session.NewSession(&aws.Config{Region: aws.String("us-west-2")})
	if err != nil {
		log.Fatalf("failed to open aws session")
	}

	// S3 and SQS Clients
	s3Client := s3.New(awsSess)
	sqsClient := sqs.New(awsSess)

	// XRay Setup
	xray.AWS(s3Client.Client)
	xray.AWS(sqsClient.Client)

	// List SQS queues
	if _, err = sqsClient.ListQueuesWithContext(subCtx, nil); err != nil {
		log.Printf("[SQS] %v", err)
	}

	// List s3 objects in bucket
	input := &s3.ListObjectsInput{Bucket: aws.String("cloudwatch-agent-integration-bucket")}
	if output, err := s3Client.ListObjectsWithContext(subCtx, input); err != nil {
		log.Printf("[S3] %v", err)
	} else {
		log.Printf("[S3] Successfully listed objects in bucket %q", *output.Name)
	}

	return root.ID
}

// OutgoingHttpCall makes an HTTP GET request to https://aws.amazon.com/ and generates an Xray Trace ID.
func (t XraySampler) OutgoingHttpCall(ctx context.Context, client http.Client) string {
	rootCtx, root := xray.BeginSegment(ctx, "xray-sample-app")
	defer root.Close(nil)
	if err := root.AddAnnotation("question", "did this work?"); err != nil {
		log.Println(err)
	}
	if err := root.AddMetadata("answer", map[string]interface{}{
		"it": "did!",
	}); err != nil {
		log.Println(err)
	}

	xrayClient := xray.Client(&client)

	req, _ := http.NewRequestWithContext(rootCtx, "GET", "https://aws.amazon.com/", nil)
	res, err := xrayClient.Do(req)
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("[HTTP] Status %d", res.StatusCode)
	}

	if err = res.Body.Close(); err != nil {
		log.Println(err)
	}

	return root.ID
}
