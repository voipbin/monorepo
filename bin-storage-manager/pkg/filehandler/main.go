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

	"cloud.google.com/go/storage"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	bucketDirectoryRecording = "recording"
	bucketDirectoryTmp       = "tmp"
	bucketDirectoryBin       = "bin" // bin project services directory.

	// downloadURLExpiration is the duration for GCS signed download URLs.
	// GCS V2 signed URLs have a hard maximum of 7 days.
	downloadURLExpiration = 7 * 24 * time.Hour
)

// FileHandler intreface for GCP bucket handler
type FileHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		ownerID uuid.UUID,
		referenceType file.ReferenceType,
		referenceID uuid.UUID,
		fileType file.Type,
		name string,
		detail string,
		filename string,
		bucketName string,
		filepath string,
	) (*file.File, error)
	Get(ctx context.Context, id uuid.UUID) (*file.File, error)
	List(ctx context.Context, token string, size uint64, filters map[file.Field]any) ([]*file.File, error)
	Delete(ctx context.Context, id uuid.UUID) (*file.File, error)
	DeleteBucketfile(ctx context.Context, bucketName string, filepath string) error

	CompressCreate(ctx context.Context, files []*file.File) (string, string, error)
	DownloadURIGet(ctx context.Context, bucketName string, filepath string, expire time.Duration) (string, string, error)
	DownloadURIRefresh(ctx context.Context, id uuid.UUID) (string, error)

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

	projectID string,
	bucketMedia string,
	bucketTmp string,
) (FileHandler, error) {
	log := logrus.WithField("func", "NewFileHandler")

	// Signing credentials are optional. Without them the handler still serves every
	// capability that does not need a signed URL (file get/list/delete, account
	// bookkeeping, the customer-deleted cascade) and file Create still persists the
	// record with an empty URIDownload. Only the signed download URL paths degrade,
	// returning a structured SIGNING_NOT_CONFIGURED error. An explicitly configured but
	// unreadable/unparsable key file is still fatal: that is broken configuration, not
	// an intentional keyless deployment.
	accessID := ""
	var privateKey []byte

	envCredPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if envCredPath == "" {
		log.Warn("GOOGLE_APPLICATION_CREDENTIALS is not set. GCS signed URL generation is disabled; download URI paths will report the signing capability as unavailable.")
	} else {
		log.Infof("Found GOOGLE_APPLICATION_CREDENTIALS at: %s", envCredPath)

		jsonContent, err := os.ReadFile(envCredPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read credential file")
		}

		conf, err := google.JWTConfigFromJSON(jsonContent)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse credential JSON")
		}

		accessID = conf.Email
		privateKey = conf.PrivateKey
	}

	promSigningAvailable.Set(0)
	if len(privateKey) > 0 && accessID != "" {
		promSigningAvailable.Set(1)
	}

	ctx := context.Background()
	client, errClient := newStorageClient(ctx, storage.NewClient, envCredPath == "")
	if errClient != nil {
		return nil, errClient
	}

	log.Debugf("Checking account. project_id: %s, bucket_media: %s, bucket_tmp: %s, access_id: %s", projectID, bucketMedia, bucketTmp, accessID)
	res := &fileHandler{
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

	return res, nil
}

// newStorageClient builds the GCS client, degrading to an unauthenticated client only
// on a deployment that declares itself keyless.
//
// GOOGLE_APPLICATION_CREDENTIALS is not only the signing key: it is also the primary
// ADC source for the storage client itself. Where the variable is unset there may be no
// credential at all, the authenticated constructor fails, and returning that error would
// put the process straight back into the boot crash-loop this handler exists to avoid.
// The fallback keeps it alive: everything that does not touch GCS (file get/list,
// account bookkeeping, the customer-deleted cascade) works normally, and the GCS paths
// fail per-request with a clear error instead.
//
// keyless gates the fallback deliberately. When a credential file IS configured, a
// client-construction failure is a real fault and stays fatal, so the pod crash-loops
// and self-heals once the underlying problem clears - silently running unauthenticated
// forever would be strictly worse there. The residual gap is a credential-less cluster
// relying on the GKE metadata server: a transient metadata failure at boot degrades it
// until a restart. That is accepted because this service's k8s manifest always sets
// GOOGLE_APPLICATION_CREDENTIALS, and the degraded state is observable through the
// storage_manager_signing_available gauge (a keyless deployment is exactly the case
// where it reads 0).
//
// newClient is injected so the fallback is testable without manipulating ambient
// credentials.
func newStorageClient(ctx context.Context, newClient func(context.Context, ...option.ClientOption) (*storage.Client, error), keyless bool) (*storage.Client, error) {
	log := logrus.WithField("func", "newStorageClient")

	res, err := newClient(ctx)
	if err == nil {
		return res, nil
	}

	if !keyless {
		return nil, errors.Wrapf(err, "failed to create client")
	}
	log.Warnf("Could not create the authenticated storage client and no credential is configured. Falling back to an unauthenticated client. err: %v", err)

	res, errFallback := newClient(ctx, option.WithoutAuthentication())
	if errFallback != nil {
		return nil, errors.Wrapf(errFallback, "failed to create client")
	}

	return res, nil
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
