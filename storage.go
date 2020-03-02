package image

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

type Config struct {
	Account       string
	AccessKey     string
	RootURL       string
	ContainerName string
}

var (
	containerName = ""
	URL           = ""
	RESOURCE_TYPE = "b"
	PERMISSION    = "r"
	API_VERSION   = "2014-02-14"
)

const (
	ExpireTime = 3600
)

func GetURL(c Config) string {
	return fmt.Sprintf(c.RootURL, c.Account, c.ContainerName)
}

func GetContainer(c Config) (azblob.ContainerURL, error) {
	credential, err := azblob.NewSharedKeyCredential(c.Account, c.AccessKey)
	if err != nil {
		return azblob.ContainerURL{}, err
	}

	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	URL, err := url.Parse(GetURL(c))
	if err != nil {
		return azblob.ContainerURL{}, err
	}

	containerURL := azblob.NewContainerURL(*URL, p)

	return containerURL, nil
}

func GetBlobURL(fileName string, withSignature bool, c Config) string {
	if fileName == "" {
		return fileName
	}

	if !withSignature {
		return fmt.Sprintf("%s/%s", GetURL(c), fileName)
	}

	timeIn := time.Now().Add(time.Second * ExpireTime)
	expiryTime := timeIn.Format("2006-01-02T15:04:05Z")
	sig := GenerateSharedAccessSignature(expiryTime, fileName, c)

	queryParams := []string{
		"se=" + url.QueryEscape(expiryTime),
		"sr=" + RESOURCE_TYPE,
		"sp=" + PERMISSION,
		"sig=" + url.QueryEscape(sig),
		"sv=" + url.QueryEscape(API_VERSION),
	}

	return fmt.Sprintf("%s/%s?%s", GetURL(c), fileName, strings.Join(queryParams, "&"))
}

func GetFileName(blobUrl string, c Config) string {
	u, err := url.Parse(blobUrl)
	if err != nil {
		return blobUrl
	}
	return strings.TrimPrefix(u.Path, "/"+c.ContainerName+"/")
}

func GenerateSharedAccessSignature(expiryTime string, fileName string, c Config) string {
	blob := fmt.Sprintf("/%s/%s/%s", c.Account, c.ContainerName, fileName)

	queryParams := []string{
		PERMISSION, // permissions
		"",
		expiryTime, // expiry
		blob,
		"",
		API_VERSION, // API version
		"", "", "", "", ""}
	toSign := strings.Join(queryParams, "\n")
	decodeAccessKey, _ := base64.StdEncoding.DecodeString(c.AccessKey)

	h := hmac.New(sha256.New, []byte(decodeAccessKey))
	h.Write([]byte(toSign))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func Upload(ctx context.Context, filePath string, buffBytes []byte, c Config) (string, error) {
	containerURL, err := GetContainer(c)
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

	return GetBlobURL(filePath, false, c), nil
}

func DeleteImage(ctx context.Context, filePath string, c Config) (string, error) {
	containerURL, err := GetContainer(c)
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

	return GetBlobURL(filePath, false, c), nil
}

func UnixNano() string {
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}

func TimebasePath() string {
	now := time.Now()
	return fmt.Sprintf("%s/%s/%s", now.Format("2006"), now.Format("01"), now.Format("02"))
}
