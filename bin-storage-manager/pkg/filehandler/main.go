package filehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package filehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	bucketDirectoryRecording = "recording"
	bucketDirectoryTmp       = "tmp"
)

// FileHandler intreface for GCP bucket handler
type FileHandler interface {
	GetDownloadURI(ctx context.Context, bucketName string, filepaths []string, expire time.Duration) (*string, *string, error)
	Delete(ctx context.Context, bucketName string, filepath string) error
	IsExist(ctx context.Context, bucketName string, filepath string) bool
}

type fileHandler struct {
	utilHandler utilhandler.UtilHandler
	client      *storage.Client

	projectID   string
	bucketMedia string
	bucketTmp   string
	accessID    string
	privateKey  []byte
}

// NewFileHandler create bucket handler
func NewFileHandler(credentialPath string, projectID string, bucketMedia string, bucketTmp string) FileHandler {

	ctx := context.Background()

	jsonKey, err := os.ReadFile(credentialPath)
	if err != nil {
		logrus.Errorf("Could not read the credential file. err: %v", err)
		return nil
	}

	// parse service account
	conf, err := google.JWTConfigFromJSON(jsonKey)
	if err != nil {
		logrus.Errorf("Could not parse the credential file. err: %v", err)
		return nil
	}

	// create client
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	h := &fileHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		client:      client,

		projectID:   projectID,
		bucketMedia: bucketMedia,
		bucketTmp:   bucketTmp,
		accessID:    conf.Email,
		privateKey:  conf.PrivateKey,
	}

	return h
}

// Init initialize the bucket
func (h *fileHandler) Init() {
	// empty
}

// getFilename returns filename
func getFilename(target string) string {
	splits := strings.Split(target, "/")
	res := splits[len(splits)-1]

	return res
}

func createZipFilepathHash(filenames []string) string {
	sort.Strings(filenames)

	tmpJoin := strings.Join(filenames, "")

	sh1 := sha1.New()
	sh1.Write([]byte(tmpJoin))
	tmp := sh1.Sum(nil)

	res := fmt.Sprintf("%s/%x.zip", bucketDirectoryTmp, tmp)

	return res
}
