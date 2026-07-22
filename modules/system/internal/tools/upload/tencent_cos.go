package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"

	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

type TencentCOS struct{}

// UploadFile upload file to COS
func (*TencentCOS) UploadFile(file io.Reader, filePath string, keepOriginalName bool) (string, error) {
	client := NewClient()
	// f, openError := file.Open()
	// if openError != nil {
	// 	shared.Logger.Errorf("function file.Open() failed: %s", openError.Error())
	// 	return "", errors.New("function file.Open() failed, err:" + openError.Error())
	// }
	// defer f.Close() // 创建文件 defer 关闭
	fileKey := fmt.Sprintf("%d%s", time.Now().Unix(), filePath)

	// Need to check what constitutes the key here.
	// Previous: global.Cfg.Admin.Oss.TencentCOS.PathPrefix + "/" + fileKey
	// This seems to be the full object key including prefix.
	fullKey := global.Cfg.System.Oss.TencentCOS.PathPrefix + "/" + fileKey

	_, err := client.Object.Put(context.Background(), fullKey, file, nil)
	if err != nil {
		panic(err)
	}
	return fullKey, nil
}

// DeleteFile delete file form COS
func (*TencentCOS) DeleteFile(key string) error {
	client := NewClient()
	name := global.Cfg.System.Oss.TencentCOS.PathPrefix + "/" + key
	_, err := client.Object.Delete(context.Background(), name)
	if err != nil {
		shared.Logger.Errorf("function bucketManager.Delete() failed: %s", err.Error())
		return errors.New("function bucketManager.Delete() failed, err:" + err.Error())
	}
	return nil
}

func (*TencentCOS) DeleteFiles(keys []string) error {
	client := NewClient()
	objects := make([]cos.Object, 0, len(keys))
	for _, key := range keys {
		if key != "" {
			objects = append(objects, cos.Object{Key: global.Cfg.System.Oss.TencentCOS.PathPrefix + "/" + key})
		}
	}
	if len(objects) == 0 {
		return nil
	}
	_, _, err := client.Object.DeleteMulti(context.Background(), &cos.ObjectDeleteMultiOptions{
		Objects: objects,
		Quiet:   true,
	})
	if err != nil {
		shared.Logger.Errorf("function Object.DeleteMulti() failed: %s", err.Error())
		return errors.New("function Object.DeleteMulti() failed, err:" + err.Error())
	}
	return nil
}

func (*TencentCOS) DownloadFile(key string) (io.ReadCloser, error) {
	client := NewClient()
	name := global.Cfg.System.Oss.TencentCOS.PathPrefix + "/" + key
	resp, err := client.Object.Get(context.Background(), name, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// NewClient init COS client
func NewClient() *cos.Client {
	urlStr, _ := url.Parse("https://" + global.Cfg.System.Oss.TencentCOS.Bucket + ".cos." + global.Cfg.System.Oss.TencentCOS.Region + ".myqcloud.com")
	baseURL := &cos.BaseURL{BucketURL: urlStr}
	client := cos.NewClient(baseURL, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  global.Cfg.System.Oss.TencentCOS.SecretID,
			SecretKey: global.Cfg.System.Oss.TencentCOS.SecretKey,
		},
	})
	return client
}
