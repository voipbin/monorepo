package filehandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-storage-manager/models/file"

	"github.com/stretchr/testify/assert"
)

func TestCompressCreate(t *testing.T) {
	ctx := context.Background()
	bucketTmp := "tmp-bucket"
	mockHash := "mock-hash-123"

	// Using an empty list of files to avoid dependencies on file.File internals (like ID type).
	// The logic flow remains the same regardless of file count.
	files := []*file.File{}

	type args struct {
		ctx          context.Context
		files        []*file.File
		bucketTmp    string
		hashFunc     func([]string) string
		isExistFunc  func(context.Context, string, string) bool
		compressFunc func(context.Context, string, []*file.File) error
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "Success - Already Exists",
			args: args{
				ctx:       ctx,
				files:     files,
				bucketTmp: bucketTmp,
				hashFunc: func(s []string) string {
					return mockHash
				},
				isExistFunc: func(ctx context.Context, bucket, filepath string) bool {
					return true // File exists
				},
				compressFunc: nil, // Should not be called
			},
			want:    bucketTmp,
			want1:   mockHash,
			wantErr: false,
		},
		{
			name: "Success - Created Successfully",
			args: args{
				ctx:       ctx,
				files:     files,
				bucketTmp: bucketTmp,
				hashFunc: func(s []string) string {
					return mockHash
				},
				isExistFunc: func() func(context.Context, string, string) bool {
					calls := 0
					return func(ctx context.Context, bucket, filepath string) bool {
						calls++
						// First call: false (not exist), Second call: true (created)
						return calls > 1
					}
				}(),
				compressFunc: func(ctx context.Context, filepath string, files []*file.File) error {
					return nil
				},
			},
			want:    bucketTmp,
			want1:   mockHash,
			wantErr: false,
		},
		{
			name: "Error - Compress Failed",
			args: args{
				ctx:       ctx,
				files:     files,
				bucketTmp: bucketTmp,
				hashFunc: func(s []string) string {
					return mockHash
				},
				isExistFunc: func(ctx context.Context, bucket, filepath string) bool {
					return false
				},
				compressFunc: func(ctx context.Context, filepath string, files []*file.File) error {
					return fmt.Errorf("compression error")
				},
			},
			want:    "",
			want1:   "",
			wantErr: true,
		},
		{
			name: "Error - Not Found After Create",
			args: args{
				ctx:       ctx,
				files:     files,
				bucketTmp: bucketTmp,
				hashFunc: func(s []string) string {
					return mockHash
				},
				isExistFunc: func(ctx context.Context, bucket, filepath string) bool {
					return false // Always returns false
				},
				compressFunc: func(ctx context.Context, filepath string, files []*file.File) error {
					return nil
				},
			},
			want:    "",
			want1:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, got1, err := compressCreate(tt.args.ctx, tt.args.files, tt.args.bucketTmp, tt.args.hashFunc, tt.args.isExistFunc, tt.args.compressFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("compressCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}
