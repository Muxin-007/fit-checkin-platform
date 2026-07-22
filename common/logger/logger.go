package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"platform/common/config"
	toolsFile "platform/common/tools/file"
)

var level zapcore.Level

func SugarLogger(cfg *config.LogConfig) (su *zap.SugaredLogger) {
	return Logger(cfg).Sugar()
}

func Logger(cfg *config.LogConfig) (logger *zap.Logger) {
	if cfg == nil {
		fmt.Println("Logger is not init.")
		os.Exit(1)
	}
	if ok, _ := toolsFile.PathExists(cfg.Director); !ok {
		fmt.Printf("create log directory: %v\n", cfg.Director)
		_ = os.Mkdir(cfg.Director, os.ModePerm)
	}
	switch strings.ToLower(cfg.Level) { // initialize config file level
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "dpanic":
		level = zap.DPanicLevel
	case "panic":
		level = zap.PanicLevel
	case "fatal":
		level = zap.FatalLevel
	default:
		level = zap.InfoLevel
	}
	logger = zap.New(getEncoderCore(cfg), zap.AddStacktrace(zapcore.DPanicLevel))
	if cfg.FileNameExt != "" {
		logger = logger.With(zap.String("from", cfg.FileNameExt))
	}
	if cfg.ShowLine {
		logger = logger.WithOptions(zap.AddCaller())
	}
	return logger
}

// getEncoderConfig 获取zapcore.EncoderConfig
func getEncoderConfig(cfg *config.LogConfig) (config zapcore.EncoderConfig) {
	config = zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  cfg.StacktraceKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     CustomTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	switch {
	case cfg.EncodeLevel == "LowercaseLevelEncoder": // 小写编码器(默认)
		config.EncodeLevel = zapcore.LowercaseLevelEncoder
	case cfg.EncodeLevel == "LowercaseColorLevelEncoder": // 小写编码器带颜色
		config.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	case cfg.EncodeLevel == "CapitalLevelEncoder": // 大写编码器
		config.EncodeLevel = zapcore.CapitalLevelEncoder
	case cfg.EncodeLevel == "CapitalColorLevelEncoder": // 大写编码器带颜色
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	default:
		config.EncodeLevel = zapcore.LowercaseLevelEncoder
	}
	return config
}

// getEncoder 获取zapcore.Encoder
func getEncoder(cfg *config.LogConfig) zapcore.Encoder {
	if cfg.Format == "json" {
		return zapcore.NewJSONEncoder(getEncoderConfig(cfg))
	}
	return zapcore.NewConsoleEncoder(getEncoderConfig(cfg))
}

// getEncoderCore 获取Encoder的zapcore.Core
func getEncoderCore(cfg *config.LogConfig) (core zapcore.Core) {
	writer, err := GetWriteSyncer(cfg) // 使用file-rotatelogs进行日志分割
	if err != nil {
		fmt.Printf("Get Write Syncer Failed err:%v", err.Error())
		return
	}
	return zapcore.NewCore(getEncoder(cfg), writer, level)
}

// CustomTimeEncoder 自定义日志输出时间格式
func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006/01/02-15:04:05.000"))
}
