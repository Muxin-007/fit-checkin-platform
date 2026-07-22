package upload

import (
	"path"
	"strings"
	"time"

	"platform/common/tools"
)

// 默认文件会处理文件名存储，可按业务需要保留原文件名。
func storageObjectName(filePath string, keepOriginalName bool) string {
	cleanName := strings.TrimLeft(path.Clean(strings.ReplaceAll(filePath, "\\", "/")), "/")
	if cleanName == "." || cleanName == "" {
		cleanName = "file"
	}
	if keepOriginalName {
		return cleanName
	}

	dir := path.Dir(cleanName)
	base := path.Base(cleanName)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		name = "file"
	}

	objectName := tools.MD5V([]byte(name)) + "_" + time.Now().Format("20060102150405") + ext
	if dir == "." || dir == "" {
		return objectName
	}
	return path.Join(dir, objectName)
}
