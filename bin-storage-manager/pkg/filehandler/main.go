package filehandler

//go:generate mockgen -package filehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-storage-manager/models/file"
	accounthandler "monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
)

const (
	bucketDirectoryRecording = "recording"
	bucketDirectoryTmp       = "tmp"
	bucketDirectoryBin       = "bin" // bin project services directory. mostly chat-manager.
)

// FileHandler intreface for GCP bucket handler
type FileHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		ownerID uuid.UUID,
		referenceType file.ReferenceType,
		referenceID uuid.UUID,
		name string,
		detail string,
		filename string,
		bucketName string,
		filepath string,
	) (*file.File, error)
	Get(ctx context.Context, id uuid.UUID) (*file.File, error)
	Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*file.File, error)
	Delete(ctx context.Context, id uuid.UUID) (*file.File, error)
	DeleteBucketfile(ctx context.Context, bucketName string, filepath string) error

	CompressCreate(ctx context.Context, files []*file.File) (string, string, error)
	DownloadURIGet(ctx context.Context, bucketName string, filepath string, expire time.Duration) (string, string, error)

	IsExist(ctx context.Context, bucketName string, filepath string) bool

	EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error
}

type fileHandler struct {
	utilHandler    utilhandler.UtilHandler
	notifyHandler  notifyhandler.NotifyHandler
	db             dbhandler.DBHandler
	accountHandler accounthandler.AccountHandler

	client *storage.Client

	projectID string

	bucketMedia string // bucket for call medias. (recording/tts/file/tmp...)
	bucketTmp   string // bucket for temporary files.

	accessID   string
	privateKey []byte
}

// NewFileHandler create bucket handler
func NewFileHandler(
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	accountHandler accounthandler.AccountHandler,

	credentialBase64 string,
	projectID string,
	bucketMedia string,
	bucketTmp string,
) FileHandler {
	log := logrus.WithField("func", "NewFileHandler")

	var client *storage.Client
	var accessID string
	var privateKey []byte
	var err error
	ctx := context.Background()

	envCredPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if envCredPath != "" {
		log.Infof("Found GOOGLE_APPLICATION_CREDENTIALS at: %s", envCredPath)

		jsonContent, err := os.ReadFile(envCredPath)
		if err != nil {
			log.Errorf("Failed to read credential file: %v", err)
			return nil
		}

		conf, err := google.JWTConfigFromJSON(jsonContent)
		if err != nil {
			log.Errorf("Failed to parse credential JSON: %v", err)
			return nil
		}

		accessID = conf.Email
		privateKey = conf.PrivateKey
		client, err = storage.NewClient(ctx)
	} else {
		log.Info("No GOOGLE_APPLICATION_CREDENTIALS, trying ADC/Metadata")

		client, err = storage.NewClient(ctx)
		privateKey = nil
		if metadata.OnGCE() {
			accessID, _ = metadata.EmailWithContext(ctx, "default")
		} else {
			log.Warn("Could not determine Service Account Email (Not on GCE/GKE)")
		}
	}

	if err != nil {
		log.Errorf("Failed to create client: %v", err)
		return nil
	}

	h := &fileHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		notifyHandler:  notifyHandler,
		db:             db,
		accountHandler: accountHandler,

		client:      client,
		projectID:   projectID,
		bucketMedia: bucketMedia,
		bucketTmp:   bucketTmp,
		accessID:    accessID,
		privateKey:  privateKey,
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
