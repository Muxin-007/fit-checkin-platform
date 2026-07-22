//go:build !windows
// +build !windows

package logger

import (
	"fmt"
	"os"
	"path"
	"time"

	zaprotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap/zapcore"

	"platform/common/config"
)

func GetWriteSyncer(cfg *config.LogConfig) (zapcore.WriteSyncer, error) {
	logPath := cfg.Director + "/" + cfg.Module
	linkName := fmt.Sprintf("%s/%s.log", logPath, cfg.Module)
	logName := path.Join(logPath, cfg.Module+"-%Y-%m-%d.log")
	if cfg.FileNameExt != "" {
		linkName = fmt.Sprintf("%s/%s-%s.log", logPath, cfg.Module, cfg.FileNameExt)
		logName = path.Join(logPath, cfg.Module+fmt.Sprintf("-%s", cfg.FileNameExt)+"-%Y-%m-%d.log")
	}
	fileWriter, err := zaprotatelogs.New(
		logName,
		zaprotatelogs.WithLinkName(linkName),
		zaprotatelogs.WithMaxAge(time.Duration(cfg.MaxAge)*time.Hour),
		zaprotatelogs.WithRotationTime(time.Duration(cfg.RotationTime)*time.Hour),
	)
	if cfg.LogInConsole {
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(fileWriter)), err
	}
	return zapcore.AddSync(fileWriter), err
}
