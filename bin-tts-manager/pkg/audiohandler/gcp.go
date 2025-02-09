package audiohandler

import (
	"context"
	"encoding/base64"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	defaultGCPEndpoint = "eu-texttospeech.googleapis.com:443"
)

func gcpGetClient(ctx context.Context, credentialBase64 string) (*texttospeech.Client, error) {
	log := logrus.WithField("func", "gcpGetClient")

	decodedCredential, err := base64.StdEncoding.DecodeString(credentialBase64)
	if err != nil {
		log.Printf("Error decoding base64 credential: %v", err)
		return nil, err
	}

	keepAliveParams := keepalive.ClientParameters{
		Time:                30 * time.Second, // Ping every 30 seconds
		Timeout:             10 * time.Second, // Wait 10 seconds for response
		PermitWithoutStream: true,             // Send pings even if there are no active streams
	}

	// create res
	res, err := texttospeech.NewClient(
		ctx,
		option.WithCredentialsJSON(decodedCredential),
		option.WithGRPCDialOption(grpc.WithKeepaliveParams(keepAliveParams)),
		option.WithEndpoint(defaultGCPEndpoint),
	)
	if err != nil {
		logrus.Errorf("Could not create a new client. err: %v", err)
		return nil, err
	}

	return res, nil
}
