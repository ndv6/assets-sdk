package file

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	Copy(ctx context.Context, filePath string) error
}

type File struct {
	Account       string
	AccessKey     string
	RootURL       string
	ContainerName string
	APIVersion    string
}

//New set account, access key, root url, container name, api version before using this library
func New(account, accessKey, rootURL, containerName, apiVersion string) IFile {
	return &File{
		Account:       account,
		AccessKey:     accessKey,
		RootURL:       rootURL,
		ContainerName: containerName,
		APIVersion:    apiVersion,
	}
}

// GetURL return string with blob_url, account and container name
func (c *File) GetURL() string {
	return fmt.Sprintf(c.RootURL, c.Account, c.ContainerName)
}

//GetContainer return container URL
func (c *File) GetContainer() (azblob.ContainerURL, error) {
	credential, err := azblob.NewSharedKeyCredential(c.Account, c.AccessKey)
	if err != nil {
		return azblob.ContainerURL{}, err
	}

	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	URL, err := url.Parse(c.GetURL())
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
func (c *File) GetBlobURL(fileName string, withSignature bool) string {
	if fileName == "" {
		return fileName
	}

	if !withSignature {
		return fmt.Sprintf("%s/%s", c.GetURL(), fileName)
	}

	timeIn := time.Now().Add(time.Second * ExpireTime)
	expiryTime := timeIn.Format("2006-01-02T15:04:05Z")
	sig := c.GenerateSharedAccessSignature(expiryTime, fileName)

	queryParams := []string{
		"se=" + url.QueryEscape(expiryTime),
		"sr=" + ResourceType,
		"sp=" + Permission,
		"sig=" + url.QueryEscape(sig),
		"sv=" + url.QueryEscape(c.APIVersion),
	}

	return fmt.Sprintf("%s/%s?%s", c.GetURL(), fileName, strings.Join(queryParams, "&"))
}

// GetFileName convert file url and return as file name.
// From "https://storage.blob.core.windows.net/container/file/image.img"
// to "file/image.img"
//
//	Example:
//	file := file.GetFileName(ctx, "https://storage.blob.core.windows.net/container/file/image.img", buffBytes)
func (c *File) GetFileName(blobUrl string) string {
	u, err := url.Parse(blobUrl)
	if err != nil {
		return blobUrl
	}
	return strings.TrimPrefix(u.Path, "/"+c.ContainerName+"/")
}

//GenerateSharedAccessSignature return access signature key
func (c *File) GenerateSharedAccessSignature(expiryTime string, fileName string) string {
	blob := fmt.Sprintf("/%s/%s/%s", c.Account, c.ContainerName, fileName)

	queryParams := []string{
		Permission, // permissions
		"",
		expiryTime, // expiry
		blob,
		"",
		c.APIVersion, // API version
		"", "", "", "", ""}
	toSign := strings.Join(queryParams, "\n")
	decodeAccessKey, _ := base64.StdEncoding.DecodeString(c.AccessKey)

	h := hmac.New(sha256.New, []byte(decodeAccessKey))
	h.Write([]byte(toSign))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Upload file to storage
//
//	Example:
//	file := file.Upload(ctx, "/file/image.img", buffBytes)
func (c *File) Upload(ctx context.Context, filePath, contentType string, buffBytes []byte) (string, error) {
	containerURL, err := c.GetContainer()
	if err != nil {
		return "", err
	}
	blobURL := containerURL.NewBlockBlobURL(filePath)
	if contentType == "" {
		contentType = http.DetectContentType(buffBytes)
	}

	_, err = blobURL.Upload(ctx,
		bytes.NewReader(buffBytes),
		azblob.BlobHTTPHeaders{ContentType: contentType},
		azblob.Metadata{}, azblob.BlobAccessConditions{})

	if err != nil {
		return "", err
	}

	return c.GetBlobURL(filePath, false), nil
}

// Delete file from storage
//
//	Example:
//	file := file.Delete(ctx, "/file/image.img")
func (c *File) Delete(ctx context.Context, filePath string) (string, error) {
	containerURL, err := c.GetContainer()
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

	return c.GetBlobURL(filePath, false), nil
}

func (c *File) GetListBlob(ctx context.Context, prefix string) (list []string, err error) {
	containerURL, err := c.GetContainer()
	if err != nil {
		return list, err
	}

	// List the blob(s) in our container; since a container may hold millions of blobs, this is done 1 segment at a time.
	for marker := (azblob.Marker{}); marker.NotDone(); { // The parens around Marker{} are required to avoid compiler error.
		listBlob, err := containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{Prefix: prefix})
		if err != nil {
			return list, err
		}
		// IMPORTANT: ListBlobs returns the start of the next segment; you MUST use this to get
		// the next segment (after processing the current result segment).
		marker = listBlob.NextMarker

		// Process the blobs returned in this result segment (if the segment is empty, the loop body won't execute)
		for _, blobInfo := range listBlob.Segment.BlobItems {
			list = append(list, blobInfo.Name)
		}
	}

	return
}

func (c *File) Download(ctx context.Context, filePath string) (file []byte, err error) {
	containerURL, err := c.GetContainer()
	if err != nil {
		return file, err
	}

	blockBlobURL := containerURL.NewBlockBlobURL(filePath)
	get, err := blockBlobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return file, err
	}

	reader := get.Body(azblob.RetryReaderOptions{})
	_ = reader.Close()

	return ioutil.ReadAll(reader)
}

func (c *File) Copy(ctx context.Context, filePath string) error {
	containerURL, err := c.GetContainer()
	if err != nil {
		return err
	}

	newBlobURL := containerURL.NewBlockBlobURL(newPath)
	res, err := newBlobURL.StartCopyFromURL(ctx, containerURL.NewBlobURL(tempPath).URL(), azblob.Metadata{}, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{})
	if err != nil {
		return err
	}
	if res.StatusCode() >= 300 {
		return errors.New("failed to copy data")
	}
}