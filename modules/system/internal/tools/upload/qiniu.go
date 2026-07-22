package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"

	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

type Qiniu struct{}

func (*Qiniu) UploadFile(file io.Reader, filePath string, keepOriginalName bool) (string, error) {
	putPolicy := storage.PutPolicy{Scope: global.Cfg.System.Oss.Qiniu.Bucket}
	mac := qbox.NewMac(global.Cfg.System.Oss.Qiniu.AccessKey, global.Cfg.System.Oss.Qiniu.SecretKey)
	upToken := putPolicy.UploadToken(mac)
	cfg := qiniuConfig()
	formUploader := storage.NewFormUploader(cfg)
	ret := storage.PutRet{}
	putExtra := storage.PutExtra{Params: map[string]string{"x:name": "github logo"}}

	// f, openError := file.Open()
	// if openError != nil {
	// 	shared.Logger.Errorf("function file.Open() failed: %s", openError.Error())
	//
	// 	return "", errors.New("function file.Open() failed, err:" + openError.Error())
	// }
	// defer f.Close()                                                  // 创建文件 defer 关闭
	fileKey := fmt.Sprintf("%d%s", time.Now().Unix(), filePath) // 文件名格式 自己可以改 建议保证唯一性
	putErr := formUploader.Put(context.Background(), &ret, upToken, fileKey, file, -1, &putExtra)
	if putErr != nil {
		shared.Logger.Errorf("function formUploader.Put() failed: %s", putErr.Error())
		return "", errors.New("function formUploader.Put() failed, err:" + putErr.Error())
	}
	return ret.Key, nil
}

func (*Qiniu) DeleteFile(key string) error {
	mac := qbox.NewMac(global.Cfg.System.Oss.Qiniu.AccessKey, global.Cfg.System.Oss.Qiniu.SecretKey)
	cfg := qiniuConfig()
	bucketManager := storage.NewBucketManager(mac, cfg)
	if err := bucketManager.Delete(global.Cfg.System.Oss.Qiniu.Bucket, key); err != nil {
		shared.Logger.Errorf("function bucketManager.Delete() failed: %s", err.Error())
		return errors.New("function bucketManager.Delete() failed, err:" + err.Error())
	}
	return nil
}

func (*Qiniu) DeleteFiles(keys []string) error {
	mac := qbox.NewMac(global.Cfg.System.Oss.Qiniu.AccessKey, global.Cfg.System.Oss.Qiniu.SecretKey)
	cfg := qiniuConfig()
	bucketManager := storage.NewBucketManager(mac, cfg)
	ops := make([]string, 0, len(keys))
	for _, key := range keys {
		if key != "" {
			ops = append(ops, storage.URIDelete(global.Cfg.System.Oss.Qiniu.Bucket, key))
		}
	}
	if len(ops) == 0 {
		return nil
	}
	rets, err := bucketManager.Batch(ops)
	if err != nil {
		shared.Logger.Errorf("function bucketManager.Batch() failed: %s", err.Error())
		return errors.New("function bucketManager.Batch() failed, err:" + err.Error())
	}
	for _, ret := range rets {
		if ret.Code >= 400 {
			return fmt.Errorf("qiniu batch delete failed, code: %d, data: %s", ret.Code, ret.Data.Error)
		}
	}
	return nil
}

func (*Qiniu) DownloadFile(key string) (io.ReadCloser, error) {
	mac := qbox.NewMac(global.Cfg.System.Oss.Qiniu.AccessKey, global.Cfg.System.Oss.Qiniu.SecretKey)
	deadline := time.Now().Add(time.Second * 3600).Unix() // 1小时有效期
	privateAccessURL := storage.MakePrivateURL(mac, global.Cfg.System.Oss.Qiniu.ImgPath, key, deadline)

	resp, err := http.Get(privateAccessURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("qiniu download failed, status: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func qiniuConfig() *storage.Config {
	cfg := storage.Config{
		UseHTTPS:      global.Cfg.System.Oss.Qiniu.UseHTTPS,
		UseCdnDomains: global.Cfg.System.Oss.Qiniu.UseCdnDomains,
	}
	switch global.Cfg.System.Oss.Qiniu.Zone { // 根据配置文件进行初始化空间对应的机房
	case "ZoneHuadong":
		cfg.Zone = &storage.ZoneHuadong
	case "ZoneHuabei":
		cfg.Zone = &storage.ZoneHuabei
	case "ZoneHuanan":
		cfg.Zone = &storage.ZoneHuanan
	case "ZoneBeimei":
		cfg.Zone = &storage.ZoneBeimei
	case "ZoneXinjiapo":
		cfg.Zone = &storage.ZoneXinjiapo
	}
	return &cfg
}
