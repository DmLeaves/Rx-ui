package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distFS embed.FS

// SetupStaticFiles 设置静态文件服务
func SetupStaticFiles(r *gin.Engine) error {
	// 获取 dist 子目录
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}

	// 静态文件服务
	staticHandler := http.FileServer(http.FS(sub))

	// 处理所有非 API 请求
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 请求返回 404
		if strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{
				"code":    404,
				"message": "API not found",
			})
			return
		}

		// 尝试提供静态文件
		file, err := sub.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			file.Close()
			staticHandler.ServeHTTP(c.Writer, c.Request)
			return
		}

		// 其他路径返回 index.html (SPA 路由)
		c.Request.URL.Path = "/"
		staticHandler.ServeHTTP(c.Writer, c.Request)
	})

	return nil
}

// GetEmbedFS 返回嵌入的文件系统
func GetEmbedFS() embed.FS {
	return distFS
}
