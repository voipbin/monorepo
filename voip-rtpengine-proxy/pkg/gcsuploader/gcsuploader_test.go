package gcsuploader

import (
	"testing"
)

func TestUploaderInterfaceCompliance(t *testing.T) {
	// Compile-time check that gcsUploader implements Uploader
	var _ Uploader = (*gcsUploader)(nil)
}
