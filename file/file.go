package file

import (
	"context"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

const (
	ExpireTime   = 3600
	ResourceType = "b"
	Permission   = "r"
)

type IFile interface {
	Upload(ctx context.Context, filePath, contentType string, buffBytes []byte) (string, error)
	Delete(ctx context.Context, filePath string) (string, error)
	GetBlobURL(fileName string, withSignature bool) string
	GetFileName(blobUrl string) string
	GetURL() string
	GetContainer() (azblob.ContainerURL, error)
	GenerateSharedAccessSignature(expiryTime string, fileName string) string
	GetListBlob(ctx context.Context, prefix string) (list []string, err error)
	Download(ctx context.Context, filePath string) ([]byte, error)
	Copy(ctx context.Context, newPath, tempPath string) error
}
