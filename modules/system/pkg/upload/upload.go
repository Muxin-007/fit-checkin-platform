package upload

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"platform/ent"
	"platform/ent/sysstoragefile"
	"platform/modules/shared"
	internalUpload "platform/modules/system/internal/tools/upload"
)

const (
	TypeFile = "file"
	TypeImg  = "img"
)

type UploadOptions struct {
	KeepOriginalName bool
	OwnerUserID      string
	Purpose          string
	Size             uint64
	Type             string
}

type UploadFileItem struct {
	File     io.Reader
	FilePath string
	Options  UploadOptions
}

// 上传文件
// filePath: 文件路径（完整路径）
func UploadFile(ctx context.Context, file io.Reader, filePath string, opts UploadOptions) (*ent.SysStorageFile, error) {
	filename := filepath.Base(filePath)
	if filename == "." || filename == "" {
		filename = "file"
	}
	fileType := opts.Type
	if fileType == "" {
		fileType = TypeFile
	}
	if fileType != TypeFile && fileType != TypeImg {
		return nil, fmt.Errorf("invalid storage file type: %s", fileType)
	}

	key, err := internalUpload.NewOss().UploadFile(file, filePath, opts.KeepOriginalName)
	if err != nil {
		return nil, err
	}

	create := shared.EntClient.SysStorageFile.Create().
		SetName(filename).
		SetTag(fileTag(filename)).
		SetSize(opts.Size).
		SetKey(key).
		SetType(sysstoragefile.Type(fileType)).
		SetNillableOwnerUserID(nonEmptyString(opts.OwnerUserID))
	if opts.Purpose != "" {
		create.SetPurpose(sysstoragefile.Purpose(opts.Purpose))
	}
	created, err := create.Save(ctx)
	if err != nil {
		_ = internalUpload.NewOss().DeleteFile(key)
		return nil, err
	}

	return created, nil
}

func UploadFiles(ctx context.Context, items []UploadFileItem) ([]*ent.SysStorageFile, error) {
	if len(items) == 0 {
		return make([]*ent.SysStorageFile, 0), nil
	}

	oss := internalUpload.NewOss()
	uploadedKeys := make([]string, 0, len(items))
	creators := make([]*ent.SysStorageFileCreate, 0, len(items))
	for _, item := range items {
		filename := filepath.Base(item.FilePath)
		if filename == "." || filename == "" {
			filename = "file"
		}
		fileType := item.Options.Type
		if fileType == "" {
			fileType = TypeFile
		}
		if fileType != TypeFile && fileType != TypeImg {
			_ = oss.DeleteFiles(uploadedKeys)
			return nil, fmt.Errorf("invalid storage file type: %s", fileType)
		}

		key, err := oss.UploadFile(item.File, item.FilePath, item.Options.KeepOriginalName)
		if err != nil {
			_ = oss.DeleteFiles(uploadedKeys)
			return nil, err
		}
		uploadedKeys = append(uploadedKeys, key)
		create := shared.EntClient.SysStorageFile.Create().
			SetName(filename).
			SetTag(fileTag(filename)).
			SetSize(item.Options.Size).
			SetKey(key).
			SetType(sysstoragefile.Type(fileType)).
			SetNillableOwnerUserID(nonEmptyString(item.Options.OwnerUserID))
		if item.Options.Purpose != "" {
			create.SetPurpose(sysstoragefile.Purpose(item.Options.Purpose))
		}
		creators = append(creators, create)
	}

	created, err := shared.EntClient.SysStorageFile.CreateBulk(creators...).Save(ctx)
	if err != nil {
		_ = oss.DeleteFiles(uploadedKeys)
		return nil, err
	}
	return created, nil
}

func nonEmptyString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func DownloadFile(ctx context.Context, id string) (io.ReadCloser, error) {
	if id == "" {
		return nil, fmt.Errorf("storage file id is empty")
	}
	file, err := shared.EntClient.SysStorageFile.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return internalUpload.NewOss().DownloadFile(file.Key)
}

func UpdateFile(ctx context.Context, id string, reader io.Reader, size uint64) (*ent.SysStorageFile, error) {
	if id == "" {
		return nil, fmt.Errorf("storage file id is empty")
	}
	file, err := shared.EntClient.SysStorageFile.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	key, err := internalUpload.NewOss().UploadFile(reader, file.Key, true)
	if err != nil {
		return nil, err
	}

	update := shared.EntClient.SysStorageFile.UpdateOneID(id).SetSize(size)
	if key != file.Key {
		update.SetKey(key)
	}
	return update.Save(ctx)
}

func DeleteFile(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("storage file key is empty")
	}

	files, err := shared.EntClient.SysStorageFile.Query().Where(sysstoragefile.KeyEQ(key)).All(ctx)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	if err = internalUpload.NewOss().DeleteFile(key); err != nil {
		return err
	}

	ids := make([]string, 0, len(files))
	for _, file := range files {
		ids = append(ids, file.ID)
	}
	if _, err = shared.EntClient.SysStorageFile.Delete().Where(sysstoragefile.IDIn(ids...)).Exec(ctx); err != nil {
		return err
	}

	return nil
}

func DeleteFiles(ctx context.Context, ids []string) error {
	cleanIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" {
			cleanIDs = append(cleanIDs, id)
		}
	}
	if len(cleanIDs) == 0 {
		return nil
	}

	files, err := shared.EntClient.SysStorageFile.Query().Where(sysstoragefile.IDIn(cleanIDs...)).All(ctx)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	fileIDs := make([]string, 0, len(files))
	keys := make([]string, 0, len(files))
	for _, file := range files {
		fileIDs = append(fileIDs, file.ID)
		keys = append(keys, file.Key)
	}
	if _, err = shared.EntClient.SysStorageFile.Delete().Where(sysstoragefile.IDIn(fileIDs...)).Exec(ctx); err != nil {
		return err
	}

	return internalUpload.NewOss().DeleteFiles(keys)
}

func EnsureStorageFile(ctx context.Context, filename string, key string, size uint64, fileType string) (*ent.SysStorageFile, error) {
	if key == "" {
		return nil, fmt.Errorf("storage file key is empty")
	}
	if fileType == "" {
		fileType = TypeFile
	}
	if fileType != TypeFile && fileType != TypeImg {
		return nil, fmt.Errorf("invalid storage file type: %s", fileType)
	}
	if size == 0 {
		actualSize, err := objectSize(key)
		if err != nil {
			return nil, err
		}
		size = actualSize
	}
	file, err := shared.EntClient.SysStorageFile.Query().Where(sysstoragefile.KeyEQ(key)).Only(ctx)
	if err == nil {
		if file.Size == 0 && size > 0 {
			return shared.EntClient.SysStorageFile.UpdateOneID(file.ID).SetSize(size).Save(ctx)
		}
		return file, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}
	if filename == "" {
		filename = filepath.Base(key)
	}
	return shared.EntClient.SysStorageFile.Create().
		SetName(filename).
		SetTag(fileTag(filename)).
		SetSize(size).
		SetKey(key).
		SetType(sysstoragefile.Type(fileType)).
		Save(ctx)
}

func objectSize(key string) (uint64, error) {
	reader, err := internalUpload.NewOss().DownloadFile(key)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	size, err := io.Copy(io.Discard, reader)
	if err != nil {
		return 0, err
	}
	return uint64(size), nil
}

func fileTag(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return strings.TrimPrefix(ext, ".")
}
