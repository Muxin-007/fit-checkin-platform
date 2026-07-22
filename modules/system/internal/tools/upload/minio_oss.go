package upload

import (
	"context"
	"errors"
	"io"
	"mime"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

var MinioClient *Minio // 优化性能，但是不支持动态配置

type Minio struct {
	Client *minio.Client
	bucket string
}

func GetMinio(endpoint, accessKeyID, secretAccessKey, bucketName string, useSSL bool) (*Minio, error) {
	if MinioClient != nil {
		return MinioClient, nil
	}
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL, // Set to true if using https
	})
	if err != nil {
		return nil, err
	}
	// 尝试创建bucket
	err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
		if errBucketExists == nil && exists {
			// log.Printf("We already own %s\n", bucketName)
		} else {
			return nil, err
		}
	}
	MinioClient = &Minio{Client: minioClient, bucket: bucketName}
	return MinioClient, nil
}

func (m *Minio) UploadFile(file io.Reader, filePath string, keepOriginalName bool) (string, error) {
	// 读取文件后缀
	ext := filepath.Ext(filePath)
	filename := storageObjectName(filePath, keepOriginalName)
	contentType := mime.TypeByExtension(ext)
	info, err := m.Client.PutObject(context.Background(), global.Cfg.System.Oss.Minio.BucketName, filename, file, -1, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		shared.Logger.Errorf("function minioClient.PutObject() Failed, err:%v", err)
		return "", errors.New("function minioClient.PutObject() Failed, err:" + err.Error())
	}
	return info.Key, nil
}

func (m *Minio) DeleteFile(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Delete the object from MinIO
	err := m.Client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
	return err
}

func (m *Minio) DeleteFiles(keys []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for _, key := range keys {
			if key != "" {
				objectsCh <- minio.ObjectInfo{Key: key}
			}
		}
	}()

	for err := range m.Client.RemoveObjects(ctx, m.bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if err.Err != nil {
			return err.Err
		}
	}
	return nil
}

func (m *Minio) DownloadFile(key string) (io.ReadCloser, error) {

	// GetObject returns a ReadSeekCloser.
	object, err := m.Client.GetObject(context.Background(), global.Cfg.System.Oss.Minio.BucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	_, err = object.Stat()
	if err != nil {
		return nil, err
	}

	return object, nil
}
