package system

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"

	"platform/modules/shared"
	"platform/modules/system/internal/config"
	"platform/modules/system/internal/domain/fitness"
	"platform/modules/system/internal/global"
	"platform/modules/system/internal/initialize"
	bootServer "platform/server"
)

type adminServer struct {
}

func init() {
	bootServer.Add(&adminServer{})
}

func (s *adminServer) Start(configPath string, routerGroup *gin.RouterGroup) {
	serverCfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v", err)
		os.Exit(1)
	}
	global.Cfg = serverCfg

	shared.Logger.Infof("==================== Launching Fitness Check-in ====================")
	shared.Logger.Infof("fitness API prefix: %s, timezone: %s", global.Cfg.System.RouterPrefix, global.Cfg.System.Timezone)

	// Initialize
	initialize.Initialize()

	// Initialize router
	fitness.InitRouter(routerGroup.Group(global.Cfg.System.RouterPrefix))
	fitness.StartReminderScheduler()
}
