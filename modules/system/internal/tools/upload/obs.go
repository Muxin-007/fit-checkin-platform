package upload

import (
	"io"
	"mime"
	"path/filepath"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/pkg/errors"

	"platform/modules/system/internal/global"
)

var HuaWeiObs = new(Obs)

type Obs struct{}

func NewHuaWeiObsClient() (client *obs.ObsClient, err error) {
	return obs.New(global.Cfg.System.Oss.HuaWeiObs.AccessKey, global.Cfg.System.Oss.HuaWeiObs.SecretKey, global.Cfg.System.Oss.HuaWeiObs.Endpoint)
}

func (o *Obs) UploadFile(file io.Reader, filePath string, keepOriginalName bool) (string, error) {
	// var open multipart.File
	// open, err := file.Open()
	// if err != nil {
	// 	return "", err
	// }
	// defer open.Close()
	filename := filePath
	input := &obs.PutObjectInput{
		PutObjectBasicInput: obs.PutObjectBasicInput{
			ObjectOperationInput: obs.ObjectOperationInput{
				Bucket: global.Cfg.System.Oss.HuaWeiObs.Bucket,
				Key:    filename,
			},
			HttpHeader: obs.HttpHeader{
				ContentType: mime.TypeByExtension(filepath.Ext(filePath)),
			},
		},
		Body: file,
	}

	var client *obs.ObsClient
	client, err := NewHuaWeiObsClient()
	if err != nil {
		return "", errors.Wrap(err, "获取华为对象存储对象失败!")
	}

	_, err = client.PutObject(input)
	if err != nil {
		return "", errors.Wrap(err, "文件上传失败!")
	}
	return filename, err
}

func (o *Obs) DeleteFile(key string) error {
	client, err := NewHuaWeiObsClient()
	if err != nil {
		return errors.Wrap(err, "获取华为对象存储对象失败!")
	}
	input := &obs.DeleteObjectInput{
		Bucket: global.Cfg.System.Oss.HuaWeiObs.Bucket,
		Key:    key,
	}
	var output *obs.DeleteObjectOutput
	output, err = client.DeleteObject(input)
	if err != nil {
		return errors.Wrapf(err, "删除对象(%s)失败!, output: %v", key, output)
	}
	return nil
}

func (o *Obs) DeleteFiles(keys []string) error {
	client, err := NewHuaWeiObsClient()
	if err != nil {
		return errors.Wrap(err, "获取华为对象存储对象失败!")
	}
	objects := make([]obs.ObjectToDelete, 0, len(keys))
	for _, key := range keys {
		if key != "" {
			objects = append(objects, obs.ObjectToDelete{Key: key})
		}
	}
	if len(objects) == 0 {
		return nil
	}
	_, err = client.DeleteObjects(&obs.DeleteObjectsInput{
		Bucket:  global.Cfg.System.Oss.HuaWeiObs.Bucket,
		Objects: objects,
		Quiet:   true,
	})
	if err != nil {
		return errors.Wrap(err, "批量删除对象失败!")
	}
	return nil
}

func (o *Obs) DownloadFile(key string) (io.ReadCloser, error) {
	client, err := NewHuaWeiObsClient()
	if err != nil {
		return nil, errors.Wrap(err, "获取华为对象存储对象失败!")
	}

	input := &obs.GetObjectInput{}
	input.Bucket = global.Cfg.System.Oss.HuaWeiObs.Bucket
	input.Key = key

	output, err := client.GetObject(input)
	if err != nil {
		return nil, err
	}

	return output.Body, nil
}
