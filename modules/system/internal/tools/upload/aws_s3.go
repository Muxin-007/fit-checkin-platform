package upload

import (
	"errors"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

type AwsS3 struct{}

func (*AwsS3) UploadFile(file io.Reader, filePath string, keepOriginalName bool) (string, error) {
	// f, openError := file.Open()
	// if openError != nil {
	// 	shared.Logger.Errorf("function file.Open() Failed, err:%v", openError)
	// 	return "", errors.New("function file.Open() Failed, err:" + openError.Error())
	// }
	// defer f.Close() // 创建文件 defer 关闭

	ext := filepath.Ext(filePath)
	filename := storageObjectName(filePath, keepOriginalName)

	session := newSession()
	uploader := s3manager.NewUploader(session)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(global.Cfg.System.Oss.AwsS3.Bucket),
		Key:         aws.String(awsS3ObjectKey(filename)),
		Body:        file,
		ContentType: aws.String(mime.TypeByExtension(ext)),
	})

	if err != nil {
		shared.Logger.Errorf("function uploader.Upload() Failed, err:%v", err)
		return "", errors.New("function uploader.Upload() Failed, err:" + err.Error())
	}

	return awsS3ObjectKey(filename), nil
}

func (*AwsS3) DeleteFile(key string) error {
	session := newSession()
	svc := s3.New(session)
	filename := awsS3ObjectKey(key)
	bucket := global.Cfg.System.Oss.AwsS3.Bucket

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

func (*AwsS3) DeleteFiles(keys []string) error {
	session := newSession()
	svc := s3.New(session)
	bucket := global.Cfg.System.Oss.AwsS3.Bucket

	objects := make([]*s3.ObjectIdentifier, 0, len(keys))
	for _, key := range keys {
		if key != "" {
			objects = append(objects, &s3.ObjectIdentifier{Key: aws.String(awsS3ObjectKey(key))})
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

func (*AwsS3) DownloadFile(key string) (io.ReadCloser, error) {
	session := newSession()
	svc := s3.New(session)

	filename := awsS3ObjectKey(key)

	output, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(global.Cfg.System.Oss.AwsS3.Bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

// newSession Create S3 session
func newSession() *session.Session {
	sess, _ := session.NewSession(&aws.Config{
		Region:           aws.String(global.Cfg.System.Oss.AwsS3.Region),
		Endpoint:         aws.String(global.Cfg.System.Oss.AwsS3.Endpoint), //minio在这里设置地址,可以兼容
		S3ForcePathStyle: aws.Bool(global.Cfg.System.Oss.AwsS3.S3ForcePathStyle),
		DisableSSL:       aws.Bool(global.Cfg.System.Oss.AwsS3.DisableSSL),
		Credentials: credentials.NewStaticCredentials(
			global.Cfg.System.Oss.AwsS3.SecretID,
			global.Cfg.System.Oss.AwsS3.SecretKey,
			"",
		),
	})
	return sess
}

func awsS3ObjectKey(key string) string {
	key = strings.TrimLeft(key, "/")
	prefix := strings.Trim(global.Cfg.System.Oss.AwsS3.PathPrefix, "/")
	if prefix == "" || key == "" {
		return key
	}
	if key == prefix || strings.HasPrefix(key, prefix+"/") {
		return key
	}
	return prefix + "/" + key
}
