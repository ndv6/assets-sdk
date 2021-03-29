package file

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type s3Manager struct {
	Session    *session.Session
	Region     string
	BucketName string
	BasePath   string
	ACL        string
}

func NewS3(session *session.Session, region, bucketName, basePath, ACL string) IFile {
	return &s3Manager{
		Session:    session,
		Region:     region,
		BucketName: bucketName,
		BasePath:   basePath,
		ACL:        ACL,
	}
}

func (s s3Manager) Upload(ctx context.Context, filePath, contentType string, buffBytes []byte) (string, error) {
	_, err := s3.New(s.Session).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(s.BucketName),
		Key:                  aws.String(fmt.Sprintf("%s%s", s.BasePath, filePath)),
		ACL:                  aws.String(s.ACL),
		Body:                 bytes.NewReader(buffBytes),
		ContentLength:        aws.Int64(int64(binary.Size(buffBytes))),
		ContentType:          aws.String(contentType),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})

	if err != nil {
		return "", err
	}

	return s.GetBlobURL(filePath, false), nil
}

func (s s3Manager) Delete(ctx context.Context, filePath string) (string, error) {
	svc := s3.New(s.Session)

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(s.BucketName), Key: aws.String(filePath)})
	if err != nil {
		return "", err
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return "", err
	}

	return s.GetBlobURL(filePath, false), nil
}

func (s s3Manager) GetBlobURL(fileName string, withSignature bool) string {
	return fmt.Sprintf("%s%s", s.GetURL(), fileName)
}

func (s s3Manager) GetFileName(blobUrl string) string {
	return filepath.Base(blobUrl)
}

func (s s3Manager) GetURL() string {
	return fmt.Sprintf("https://%s.%s.amazonaws.com/%s", s.BucketName, s.Region, s.BasePath)
}

func (s s3Manager) GetContainer() (azblob.ContainerURL, error) {
	panic("not implement")
}

func (s s3Manager) GenerateSharedAccessSignature(expiryTime string, fileName string) string {
	panic("not implement")
}

func (s s3Manager) GetListBlob(ctx context.Context, prefix string) (list []string, err error) {
	svc := s3.New(s.Session)

	// Get the list of items
	var resp *s3.ListObjectsV2Output
	resp, err = svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(s.BucketName), Prefix: aws.String(prefix)})
	if err != nil {
		return
	}

	for _, item := range resp.Contents {
		list = append(list, *item.Key)
	}

	return
}

func (s s3Manager) Download(ctx context.Context, filePath string) (file []byte, err error) {
	downloader := s3manager.NewDownloader(s.Session)

	filename := filepath.Base(filePath)
	tempFile, err := os.Create(filename)
	if err != nil {
		return file, err
	}
	defer os.Remove(filename)
	defer tempFile.Close()

	_, err = downloader.DownloadWithContext(ctx, tempFile,
		&s3.GetObjectInput{
			Bucket: aws.String(s.BucketName),
			Key:    aws.String(filePath),
		})
	if err != nil {
		return file, err
	}

	return ioutil.ReadAll(tempFile)
}

func (s s3Manager) Copy(ctx context.Context, newPath, tempPath string) (err error) {
	svc := s3.New(s.Session)

	_, err = svc.CopyObject(&s3.CopyObjectInput{Bucket: aws.String(s.BucketName),
		CopySource: aws.String(url.PathEscape(tempPath)), Key: aws.String(newPath)})
	if err != nil {
		return
	}

	// Wait to see if the item got copied
	err = svc.WaitUntilObjectExists(&s3.HeadObjectInput{Bucket: aws.String(s.BucketName), Key: aws.String(newPath)})
	return
}
