package upload

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

var mu sync.Mutex

type Local struct{}

func (*Local) UploadFile(file io.Reader, filePath string, keepOriginalName bool) (string, error) {
	filePath = storageObjectName(filePath, keepOriginalName)
	// try to create this path
	p := filepath.Join(global.Cfg.System.Oss.Local.StorePath, filePath)
	mkdirErr := os.MkdirAll(filepath.Dir(p), os.ModePerm)
	if mkdirErr != nil {
		shared.Logger.Errorf("function os.MkdirAll() failed: %s", mkdirErr.Error())
		return "", errors.New("function os.MkdirAll() failed, err:" + mkdirErr.Error())
	}

	// f, openError := file.Open() // read file
	// if openError != nil {
	// 	shared.Logger.Errorf("function file.Open() failed: %s", openError.Error())
	// 	return "", errors.New("function file.Open() failed, err:" + openError.Error())
	// }
	// defer f.Close() // create file defer close

	out, createErr := os.Create(p)
	if createErr != nil {
		shared.Logger.Errorf("function os.Create() failed: %s", createErr.Error())

		return "", errors.New("function os.Create() failed, err:" + createErr.Error())
	}
	defer out.Close() // create file defer close

	_, copyErr := io.Copy(out, file) // transfer (copy) file
	if copyErr != nil {
		shared.Logger.Errorf("function io.Copy() failed: %s", copyErr.Error())
		return "", errors.New("function io.Copy() failed, err:" + copyErr.Error())
	}
	return filePath, nil
}

func (*Local) DeleteFile(key string) error {
	// check key is empty
	if key == "" {
		return errors.New("key不能为空")
	}

	// check key is invalid
	if strings.Contains(key, "..") || strings.ContainsAny(key, `:*?"<>|`) {
		return errors.New("非法的key")
	}

	p := filepath.Join(global.Cfg.System.Oss.Local.StorePath, key)

	// check file is exist
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return errors.New("文件不存在")
	}

	// use
	mu.Lock()
	defer mu.Unlock()

	err := os.Remove(p)
	if err != nil {
		return errors.New("文件删除失败: " + err.Error())
	}

	return nil
}

func (l *Local) DeleteFiles(keys []string) error {
	return deleteFiles(l, keys)
}

func (*Local) DownloadFile(key string) (io.ReadCloser, error) {
	p := filepath.Join(global.Cfg.System.Oss.Local.StorePath, key)
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	return f, nil
}
