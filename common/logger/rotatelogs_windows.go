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
	// logPath := logDirector + "/" + LogModule
	// linkName := fmt.Sprintf("%s/%s.log", logPath, LogModule)

	logName := path.Join(cfg.Module + "-%Y-%m-%d.log")
	if cfg.FileNameExt != "" {
		logName = path.Join(cfg.Module + fmt.Sprintf("-%s", cfg.FileNameExt) + "-%Y-%m-%d.log")
	}

	fileWriter, err := zaprotatelogs.New(
		logName,
		zaprotatelogs.WithMaxAge(time.Duration(cfg.MaxAge)*time.Hour),
		zaprotatelogs.WithRotationTime(time.Duration(cfg.RotationTime)*time.Hour),
	)
	if cfg.LogInConsole {
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(fileWriter)), err
	}
	return zapcore.AddSync(fileWriter), err
}
