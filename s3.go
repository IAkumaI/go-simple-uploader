package simpleuploader

import (
	"io"
	"log"
	"os"
	"strings"
	"time"
)

import (
	"github.com/IAkumaI/retry"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Uploader struct {
	client     *s3.S3
	bucket     string
	dir        string
	pathPrefix string
	urlPrefix  string
}

// NewS3 создает настроенный S3Uploader uploader
func NewS3(config *aws.Config, bucket string, dir string, pathPrefix string, urlPrefix string) Uploader {
	sess, err := session.NewSession(config)
	if err != nil {
		panic(err)
	}

	return &S3Uploader{
		client:     s3.New(sess),
		bucket:     bucket,
		dir:        dir,
		pathPrefix: pathPrefix,
		urlPrefix:  urlPrefix,
	}
}

func (uploader *S3Uploader) Upload(file *os.File, name string) (string, error) {
	result := ""

	objKey := uploader.pathPrefix + "/" + name
	if uploader.dir != "" {
		objKey = strings.TrimRight(uploader.dir, "/") + "/" + strings.TrimLeft(objKey, "/")
	}

	err := retry.Do(10, func(retryCount int) error {
		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			log.Println("Can not seek to file start")
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}

		_, err = uploader.client.PutObject(&s3.PutObjectInput{
			Body:               file,
			Bucket:             aws.String(uploader.bucket),
			Key:                aws.String(objKey),
			ContentDisposition: aws.String("attachment"),
		})

		if err != nil {
			log.Printf("Can not upload to S3 %s: %v\n", name, err)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}

		result = uploader.urlPrefix + uploader.pathPrefix + "/" + name
		return nil
	})
	if err != nil {
		return "", err
	}
	return result, nil
}
