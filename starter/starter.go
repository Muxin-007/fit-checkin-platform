package starter

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"

	"platform/modules/shared"
	_ "platform/modules/system"

	"platform/common/logger"
	"platform/config"
	"platform/server"
)

func Start(configPath string) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v", err)
		os.Exit(1)
	}

	// Init logger
	shared.Logger = logger.SugarLogger(&cfg.Log)

	if cfg.Env == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	routerGroup := router.Group(cfg.RouterPrefix)

	server.Boot(configPath, routerGroup)
	if err = router.Run(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		fmt.Printf("Failed to serve HTTP: %v\n", err)
		os.Exit(1)
	}
}
