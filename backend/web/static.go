package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var embeddedFS embed.FS

// RegisterStaticRoutes serves embedded frontend assets and falls back to index.html for SPA routing
func RegisterStaticRoutes(r *gin.Engine) {
	distFS, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Don't intercept API routes
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/health") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "API endpoint not found"})
			return
		}

		filePath := strings.TrimPrefix(path, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		// Check if file exists in embedded filesystem
		f, err := distFS.Open(filePath)
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Fallback to index.html for SPA client-side routing
		indexFile, err := distFS.Open("index.html")
		if err == nil {
			defer indexFile.Close()
			content, readErr := io.ReadAll(indexFile)
			if readErr == nil {
				c.Data(http.StatusOK, "text/html; charset=utf-8", content)
				return
			}
		}

		c.String(http.StatusNotFound, "Page not found")
	})
}
