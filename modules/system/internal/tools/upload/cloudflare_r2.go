package upload

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

type CloudflareR2 struct{}

func (c *CloudflareR2) UploadFile(file io.Reader, filePath string, keepOriginalName bool) (string, error) {
	session := c.newSession()
	client := s3manager.NewUploader(session)

	fileKey := fmt.Sprintf("%d_%s", time.Now().Unix(), filePath)
	fileName := fmt.Sprintf("%s/%s", global.Cfg.System.Oss.CloudflareR2.Path, fileKey)
	// f, openError := file.Open()
	// if openError != nil {
	// 	shared.Logger.Errorf("function file.Open() failed: %s", openError.Error())
	// 	return "", errors.New("function file.Open() failed, err:" + openError.Error())
	// }
	// defer f.Close() // 创建文件 defer 关闭

	input := &s3manager.UploadInput{
		Bucket: aws.String(global.Cfg.System.Oss.CloudflareR2.Bucket),
		Key:    aws.String(fileName),
		Body:   file,
	}

	_, err := client.Upload(input)
	if err != nil {
		shared.Logger.Errorf("function uploader.Upload() failed: %s", err.Error())
		return "", err
	}

	return fileName, nil
}

func (c *CloudflareR2) DeleteFile(key string) error {
	session := newSession()
	svc := s3.New(session)
	filename := global.Cfg.System.Oss.CloudflareR2.Path + "/" + key
	bucket := global.Cfg.System.Oss.CloudflareR2.Bucket

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		shared.Logger.Errorf("function svc.DeleteObject() failed: %s", err.Error())
		return errors.New("function svc.DeleteObject() failed, err:" + err.Error())
	}

	_ = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})
	return nil
}

func (c *CloudflareR2) DeleteFiles(keys []string) error {
	session := c.newSession()
	svc := s3.New(session)
	bucket := global.Cfg.System.Oss.CloudflareR2.Bucket

	objects := make([]*s3.ObjectIdentifier, 0, len(keys))
	for _, key := range keys {
		if key != "" {
			objects = append(objects, &s3.ObjectIdentifier{Key: aws.String(global.Cfg.System.Oss.CloudflareR2.Path + "/" + key)})
		}
	}
	if len(objects) == 0 {
		return nil
	}

	_, err := svc.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &s3.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})
	if err != nil {
		shared.Logger.Errorf("function svc.DeleteObjects() failed: %s", err.Error())
		return errors.New("function svc.DeleteObjects() failed, err:" + err.Error())
	}
	return nil
}

func (c *CloudflareR2) DownloadFile(key string) (io.ReadCloser, error) {
	session := c.newSession()
	svc := s3.New(session)

	// Assume logic mirrors AwsS3 which mirrors previous implementation
	filename := global.Cfg.System.Oss.CloudflareR2.Path + "/" + key

	output, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(global.Cfg.System.Oss.CloudflareR2.Bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

func (*CloudflareR2) newSession() *session.Session {
	endpoint := fmt.Sprintf("%s.r2.cloudflarestorage.com", global.Cfg.System.Oss.CloudflareR2.AccountID)

	return session.Must(session.NewSession(&aws.Config{
		Region:   aws.String("auto"),
		Endpoint: aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials(
			global.Cfg.System.Oss.CloudflareR2.AccessKeyID,
			global.Cfg.System.Oss.CloudflareR2.SecretAccessKey,
			"",
		),
	}))
}
