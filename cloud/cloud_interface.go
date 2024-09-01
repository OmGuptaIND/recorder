package cloud

import (
	"context"

	"github.com/OmGuptaIND/config"
)

// GetStore retrieves the store from the context, if ctx is nil it returns the global store.
func GetClient(ctx *context.Context) CloudClient {
	ctxStore, _ := (*ctx).Value(config.CloudClientKey).(CloudClient)

	return ctxStore
}

type CloudClient interface {
	CreateMultipartUpload(storagePath *string) (*string, error)
	UploadPart(input *CloudUploadPartInput) (*CloudUploadPartReponse, error)
	CompletePartUpload(input *CloudUploadPartInput) (*CloudUploadPartCompleted, error)
	UploadFile(fileName *string, filePath string) error
	DownloadFile(fileName *string, downloadPath string) error
}

type CloudUploadPartInput struct {
	PartNumber  int
	UploadId    string
	Buffer      *[]byte
	StoragePath *string
	Parts       *[]*CloudUploadPartReponse
}

type CloudUploadPartReponse struct {
	ETag       *string
	PartNumber *int64
}

type CloudUploadPartCompleted struct {
	Recording_Url *string
}
