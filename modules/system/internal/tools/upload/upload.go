package upload

import (
	"io"

	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

type OSS interface {
	UploadFile(file io.Reader, filePath string, keepOriginalName bool) (string, error)
	DeleteFile(key string) error
	DeleteFiles(keys []string) error
	DownloadFile(key string) (io.ReadCloser, error)
}

type singleFileDeleter interface {
	DeleteFile(key string) error
}

func deleteFiles(deleter singleFileDeleter, keys []string) error {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if err := deleter.DeleteFile(key); err != nil {
			return err
		}
	}
	return nil
}

func NewOss() OSS {
	switch global.Cfg.System.Oss.Type {
	case "local":
		return &Local{}
	case "qiniu":
		return &Qiniu{}
	case "tencent-cos":
		return &TencentCOS{}
	case "aliyun-oss":
		return &AliyunOSS{}
	case "huawei-obs":
		return HuaWeiObs
	case "aws-s3":
		return &AwsS3{}
	case "cloudflare-r2":
		return &CloudflareR2{}
	case "minio":
		minioClient, err := GetMinio(global.Cfg.System.Oss.Minio.Endpoint, global.Cfg.System.Oss.Minio.AccessKeyId, global.Cfg.System.Oss.Minio.AccessKeySecret, global.Cfg.System.Oss.Minio.BucketName, global.Cfg.System.Oss.Minio.UseSSL)
		if err != nil {
			shared.Logger.Warn("you configured to use minio, but initialization failed, please check the availability or security configuration of minio: " + err.Error())
			panic("minio initialization failed")
		}
		return minioClient
	default:
		return &Local{}
	}
}
