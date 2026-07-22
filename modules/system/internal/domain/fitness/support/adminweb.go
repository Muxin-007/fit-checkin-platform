package support

import (
	"embed"
	"io/fs"
	"net/http"
	"path"

	"github.com/gin-gonic/gin"
)

//go:embed admin/*
var adminAssets embed.FS

func ServeAdminWeb(c *gin.Context) {
	name := c.Param("asset")
	if name == "" || name == "/" {
		name = "index.html"
	} else {
		name = path.Base(name)
	}
	assets, err := fs.Sub(adminAssets, "admin")
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	content, err := fs.ReadFile(assets, name)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	switch path.Ext(name) {
	case ".css":
		c.Header("Content-Type", "text/css; charset=utf-8")
	case ".js":
		c.Header("Content-Type", "application/javascript; charset=utf-8")
	default:
		c.Header("Content-Type", "text/html; charset=utf-8")
	}
	c.Data(http.StatusOK, c.Writer.Header().Get("Content-Type"), content)
}
