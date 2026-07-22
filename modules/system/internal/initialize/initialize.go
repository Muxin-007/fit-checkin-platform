package initialize

import (
	"context"
	"os"

	"go.uber.org/zap/zapcore"

	"platform/common/db"
	"platform/common/db/redis"
	"platform/common/tools/id"
	"platform/ent"
	"platform/ent/migrate"
	_ "platform/ent/runtime" // 初始化ent事件
	"platform/modules/shared"
	"platform/modules/system/internal/global"
)

func Initialize() {
	initSnowflake()
	initDb()
	global.Redis = redis.InitRedis(global.Cfg.System.RedisCfg)
}

// initDb init db
func initDb() {
	// create database
	db.CreateDatabase(global.Cfg.System.DbConfig)

	dsn, err := global.Cfg.System.DbConfig.GetDSN(true)
	if err != nil {
		shared.Logger.Errorf("init db client failed: %s", err)
		os.Exit(1)
	}
	option := make([]ent.Option, 0)
	if shared.Logger.Level() == zapcore.DebugLevel {
		option = append(option, ent.Debug())
	}

	entClient, err := ent.Open(string(global.Cfg.System.DbConfig.Driver), dsn, option...)
	if err != nil {
		shared.Logger.Errorf("init ent client failed: %s", err)
		os.Exit(1)
	}
	shared.EntClient = entClient

	err = entClient.Schema.Create(context.Background(), migrate.WithForeignKeys(false))
	if err != nil {
		shared.Logger.Errorf("init ent schema failed: %s", err)
		os.Exit(1)
	}

}

// initSnowflake init snowflake
func initSnowflake() {
	id.InitSonyflake(uint16(global.Cfg.System.NodeID))
}
