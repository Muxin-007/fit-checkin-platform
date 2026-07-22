package server

import (
	"sync"

	"github.com/gin-gonic/gin"
)

var activeServers = struct {
	sync.RWMutex
	s []Server
}{}

type Server interface {
	Start(configPath string, routerGroup *gin.RouterGroup)
}

func Add(s Server) {
	activeServers.Lock()
	defer activeServers.Unlock()

	activeServers.s = append(activeServers.s, s)
}

func Boot(configPath string, routerGroup *gin.RouterGroup) {
	for _, registration := range activeServers.s {
		registration.Start(configPath, routerGroup)
	}
}
