package streaminghandler

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	"github.com/gofrs/uuid"

	"monorepo/bin-transcribe-manager/models/streaming"
)

// newUnreachableAWSClient returns a client whose endpoint is a loopback port nothing listens on
// and that never retries, so a call fails right away without leaving the machine.
func newUnreachableAWSClient(t *testing.T) *transcribestreaming.Client {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	return transcribestreaming.New(transcribestreaming.Options{
		Region:       defaultAWSRegion,
		BaseEndpoint: aws.String("http://" + addr),
		Credentials:  credentials.NewStaticCredentialsProvider("test_access_key", "test_secret_key", ""),
		Retryer:      aws.NopRetryer{},
	})
}

func Test_awsInit_unreachableEndpoint(t *testing.T) {
	h := &streamingHandler{awsClient: newUnreachableAWSClient(t)}
	st := &streaming.Streaming{
		ID:           uuid.FromStringOrNil("7b0e5a40-0000-11ee-8c10-000000000001"),
		TranscribeID: uuid.FromStringOrNil("7b0e5a40-0000-11ee-8c10-000000000002"),
		Language:     "en-US",
	}

	res, err := h.awsInit(context.Background(), st)
	if err == nil {
		t.Fatalf("awsInit() expected an error for an unreachable endpoint")
	}
	if res != nil {
		t.Errorf("awsInit() expected a nil result on error. got: %v", res)
	}
}

func Test_awsRun(t *testing.T) {
	st := &streaming.Streaming{
		ID:           uuid.FromStringOrNil("7b0e5a40-0000-11ee-8c10-000000000003"),
		TranscribeID: uuid.FromStringOrNil("7b0e5a40-0000-11ee-8c10-000000000004"),
		Language:     "en-US",
	}

	t.Run("client not initialized", func(t *testing.T) {
		h := &streamingHandler{}

		err := h.awsRun(st)
		if err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("awsRun() expected the not-initialized error. got: %v", err)
		}
	})

	t.Run("stream start failure is returned", func(t *testing.T) {
		h := &streamingHandler{awsClient: newUnreachableAWSClient(t)}

		if err := h.awsRun(st); err == nil {
			t.Errorf("awsRun() expected the stream start error")
		}
	})
}
