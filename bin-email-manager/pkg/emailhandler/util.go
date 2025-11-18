package emailhandler

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultDownloadTimeout = 30 * time.Second
	defaultMaxDownloadSize = 50 * 1024 * 1024 // 50 MB
)

var httpClient = &http.Client{
	Timeout: defaultDownloadTimeout,
}

func download(ctx context.Context, downloadURI string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURI, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create request")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to download file")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file. status=%d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, defaultMaxDownloadSize))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read response body")
	}

	return data, nil
}

// downloadToBase64 downloads the file from the given uri and returns the base64 encoded string
func downloadToBase64(ctx context.Context, downloadURI string) (string, error) {

	tmp, err := download(ctx, downloadURI)
	if err != nil {
		return "", errors.Wrapf(err, "could not download file")
	}

	return base64.StdEncoding.EncodeToString(tmp), nil
}
