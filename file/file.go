package file

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ndv6/goconf"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

const (
	ExpireTime = 3600
)

var (
	Account       = goconf.GetString("azure.storage.account")
	AccessKey     = goconf.GetString("azure.storage.access_key")
	RootURL       = goconf.GetString("azure.storage.blob_url")
	ContainerName = goconf.GetString("azure.storage.container_name")

	containerName = ""
	URL           = ""
	RESOURCE_TYPE = "b"
	PERMISSION    = "r"
	API_VERSION   = "2014-02-14"
)

// GetURL return string with blob_url, account and container name
func GetURL() string {
	return fmt.Sprintf(RootURL, Account, ContainerName)
}

//GetContainer return container URL
func GetContainer() (azblob.ContainerURL, error) {
	credential, err := azblob.NewSharedKeyCredential(Account, AccessKey)
	if err != nil {
		return azblob.ContainerURL{}, err
	}

	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	URL, err := url.Parse(GetURL())
	if err != nil {
		return azblob.ContainerURL{}, err
	}

	containerURL := azblob.NewContainerURL(*URL, p)

	return containerURL, nil
}

// GetBlobURL convert file name and return as file url.
// From "file/image.img"
// to "https://storage.blob.core.windows.net/container/file/image.img"
//
// if withSignature set to true, will return file url with access signature key
// if withSignature set to false, will return file url without access signature key
//
//	Example:
//	url := file.GetFileName("file/image.img", false)
//
func GetBlobURL(fileName string, withSignature bool) string {
	if fileName == "" {
		return fileName
	}

	if !withSignature {
		return fmt.Sprintf("%s/%s", GetURL(), fileName)
	}

	timeIn := time.Now().Add(time.Second * ExpireTime)
	expiryTime := timeIn.Format("2006-01-02T15:04:05Z")
	sig := GenerateSharedAccessSignature(expiryTime, fileName)

	queryParams := []string{
		"se=" + url.QueryEscape(expiryTime),
		"sr=" + RESOURCE_TYPE,
		"sp=" + PERMISSION,
		"sig=" + url.QueryEscape(sig),
		"sv=" + url.QueryEscape(API_VERSION),
	}

	return fmt.Sprintf("%s/%s?%s", GetURL(), fileName, strings.Join(queryParams, "&"))
}

// GetFileName convert file url and return as file name.
// From "https://storage.blob.core.windows.net/container/file/image.img"
// to "file/image.img"
//
//	Example:
//	file := file.GetFileName(ctx, "https://storage.blob.core.windows.net/container/file/image.img", buffBytes)
func GetFileName(blobUrl string) string {
	u, err := url.Parse(blobUrl)
	if err != nil {
		return blobUrl
	}
	return strings.TrimPrefix(u.Path, "/"+ContainerName+"/")
}

//GenerateSharedAccessSignature return access signature key
func GenerateSharedAccessSignature(expiryTime string, fileName string) string {
	blob := fmt.Sprintf("/%s/%s/%s", Account, ContainerName, fileName)

	queryParams := []string{
		PERMISSION, // permissions
		"",
		expiryTime, // expiry
		blob,
		"",
		API_VERSION, // API version
		"", "", "", "", ""}
	toSign := strings.Join(queryParams, "\n")
	decodeAccessKey, _ := base64.StdEncoding.DecodeString(AccessKey)

	h := hmac.New(sha256.New, []byte(decodeAccessKey))
	h.Write([]byte(toSign))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Upload file to storage
//
//	Example:
//	file := file.Upload(ctx, "/file/image.img", buffBytes)
func Upload(ctx context.Context, filePath string, buffBytes []byte) (string, error) {
	containerURL, err := GetContainer()
	if err != nil {
		return "", err
	}
	blobURL := containerURL.NewBlockBlobURL(filePath)
	contentType := http.DetectContentType(buffBytes)

	_, err = blobURL.Upload(ctx,
		bytes.NewReader(buffBytes),
		azblob.BlobHTTPHeaders{ContentType: contentType},
		azblob.Metadata{}, azblob.BlobAccessConditions{})

	if err != nil {
		return "", err
	}

	return GetBlobURL(filePath, false), nil
}

// Delete file from storage
//
//	Example:
//	file := file.Delete(ctx, "/file/image.img")
func Delete(ctx context.Context, filePath string) (string, error) {
	containerURL, err := GetContainer()
	if err != nil {
		return "", err
	}
	blobURL := containerURL.NewBlockBlobURL(filePath)

	_, err = blobURL.Delete(ctx,
		azblob.DeleteSnapshotsOptionInclude,
		azblob.BlobAccessConditions{})

	if err != nil {
		return "", err
	}

	return GetBlobURL(filePath, false), nil
}
