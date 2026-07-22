package upload

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"platform/common/tools" // Added import

	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

type AliyunOSS struct{}

func (*AliyunOSS) UploadFile(file io.Reader, filePath string, keepOriginalName bool) (string, error) {
	bucket, err := NewBucket()
	if err != nil {
		shared.Logger.Errorf("function AliyunOSS.NewBucket() Failed, err:%v", err)
		return "", errors.New("function AliyunOSS.NewBucket() Failed, err:" + err.Error())
	}

	// 读取文件后缀
	ext := filepath.Ext(filePath)

	// 读取文件名并加密
	name := strings.TrimSuffix(filePath, ext)
	name = tools.MD5V([]byte(name))

	// 拼接新文件名
	yunFileTmpPath := global.Cfg.System.Oss.AliyunOSS.BasePath + "/" + "uploads" + "/" + time.Now().Format("2006-01-02") + "/" + name + ext

	// f, openError := file.Open()
	// if openError != nil {
	// 	shared.Logger.Errorf("function file.Open() Failed, err:%v", openError)
	// 	return "", errors.New("function file.Open() Failed, err:" + openError.Error())
	// }
	// defer f.Close() // 创建文件 defer 关闭

	// 上传文件流。
	err = bucket.PutObject(yunFileTmpPath, file)
	if err != nil {
		shared.Logger.Errorf("function formUploader.Put() Failed, err:%v", err)
		return "", errors.New("function formUploader.Put() Failed, err:" + err.Error())
	}

	return yunFileTmpPath, nil
}

func (*AliyunOSS) DeleteFile(key string) error {
	bucket, err := NewBucket()
	if err != nil {
		shared.Logger.Errorf("function AliyunOSS.NewBucket() Failed: %s", err.Error())
		return errors.New("function AliyunOSS.NewBucket() Failed, err:" + err.Error())
	}

	// 删除单个文件。objectName表示删除OSS文件时需要指定包含文件后缀在内的完整路径，例如abc/efg/123.jpg。
	// 如需删除文件夹，请将objectName设置为对应的文件夹名称。如果文件夹非空，则需要将文件夹下的所有object删除后才能删除该文件夹。
	err = bucket.DeleteObject(key)
	if err != nil {
		shared.Logger.Errorf("function bucketManager.Delete() failed: %s", err.Error())
		return errors.New("function bucketManager.Delete() failed, err:" + err.Error())
	}

	return nil
}

func (*AliyunOSS) DeleteFiles(keys []string) error {
	bucket, err := NewBucket()
	if err != nil {
		shared.Logger.Errorf("function AliyunOSS.NewBucket() Failed: %s", err.Error())
		return errors.New("function AliyunOSS.NewBucket() Failed, err:" + err.Error())
	}

	filteredKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		if key != "" {
			filteredKeys = append(filteredKeys, key)
		}
	}
	if len(filteredKeys) == 0 {
		return nil
	}

	_, err = bucket.DeleteObjects(filteredKeys)
	if err != nil {
		shared.Logger.Errorf("function bucket.DeleteObjects() failed: %s", err.Error())
		return errors.New("function bucket.DeleteObjects() failed, err:" + err.Error())
	}

	return nil
}

func (*AliyunOSS) DownloadFile(key string) (io.ReadCloser, error) {
	bucket, err := NewBucket()
	if err != nil {
		return nil, errors.New("function AliyunOSS.NewBucket() Failed, err:" + err.Error())
	}

	// 下载文件到流。
	body, err := bucket.GetObject(key)
	if err != nil {
		return nil, errors.New("function bucket.GetObject() Failed, err:" + err.Error())
	}
	return body, nil
}

func NewBucket() (*oss.Bucket, error) {
	// 创建OSSClient实例。
	client, err := oss.New(global.Cfg.System.Oss.AliyunOSS.Endpoint, global.Cfg.System.Oss.AliyunOSS.AccessKeyId, global.Cfg.System.Oss.AliyunOSS.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	// 获取存储空间。
	bucket, err := client.Bucket(global.Cfg.System.Oss.AliyunOSS.BucketName)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}
