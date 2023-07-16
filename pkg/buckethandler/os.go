package buckethandler

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

// OSFileExist return the true if the given target is exists in the os filesystem
func (h *bucketHandler) OSFileExist(ctx context.Context, target string) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":   "OSFileExist",
		"target": target,
	})

	fileInfo, err := os.Stat(target)
	if err != nil {
		return false
	}
	log.WithField("file", fileInfo).Infof("The target file is exsits. target: %s", target)

	return true
}

// OSGetFilepath return the filepath of the given target filename
func (h *bucketHandler) OSGetFilepath(ctx context.Context, target string) string {
	res := fmt.Sprintf("%s/%s", h.osBucketDirectory, target)
	return res
}

// OSGetMediaFilepath return the filepath for the media of the given target filename
func (h *bucketHandler) OSGetMediaFilepath(ctx context.Context, target string) string {
	res := fmt.Sprintf("http://%s/%s", h.osLocalAddress, target)
	return res
}
