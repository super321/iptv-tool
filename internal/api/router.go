package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/publish"
	"iptv-tool-v2/internal/task"
	"iptv-tool-v2/pkg/auth"
)

// SetupRouter creates and configures the Gin router with all routes
func SetupRouter(scheduler *task.Scheduler, logoDir string, frontendFS fs.FS) *gin.Engine {
	r := gin.Default()

	// --- Public routes (no auth required) ---

	// System initialization
	systemCtrl := NewSystemController()
	r.GET("/api/init/status", systemCtrl.CheckInit)
	r.POST("/api/init", systemCtrl.Init)
	r.POST("/api/login", systemCtrl.Login)
	r.GET("/api/captcha", systemCtrl.GetCaptcha)

	// Logo static files (public access for player clients)
	r.Static("/logo", logoDir)

	// Published subscription endpoints (public access for player clients)
	r.GET("/sub/live/:path", publish.LiveHandler)
	r.GET("/sub/epg/:path", publish.EPGHandler)

	// --- Settings (public, no auth) ---
	r.GET("/api/settings/cron-options", GetCronOptions)
	r.GET("/api/settings/epg-strategies", GetEPGStrategies)

	// --- Protected routes (JWT auth required) ---
	authorized := r.Group("/api")
	authorized.Use(auth.JWTAuthMiddleware())
	{
		// User management
		authorized.POST("/user/password", systemCtrl.ChangePassword)
		authorized.POST("/crack-key", systemCtrl.CrackKey)

		// Live Sources CRUD
		liveCtrl := NewLiveSourceController(scheduler)
		authorized.GET("/live-sources", liveCtrl.List)
		authorized.GET("/live-sources/unlinked-iptv", liveCtrl.UnlinkedIPTV)
		authorized.GET("/live-sources/:id", liveCtrl.Get)
		authorized.POST("/live-sources", liveCtrl.Create)
		authorized.PUT("/live-sources/:id", liveCtrl.Update)
		authorized.DELETE("/live-sources/:id", liveCtrl.Delete)
		authorized.POST("/live-sources/:id/trigger", liveCtrl.Trigger)
		authorized.GET("/live-sources/:id/channels", liveCtrl.GetChannels)

		// EPG Sources CRUD
		epgCtrl := NewEPGSourceController(scheduler)
		authorized.GET("/epg-sources", epgCtrl.List)
		authorized.GET("/epg-sources/:id", epgCtrl.Get)
		authorized.POST("/epg-sources", epgCtrl.Create)
		authorized.PUT("/epg-sources/:id", epgCtrl.Update)
		authorized.DELETE("/epg-sources/:id", epgCtrl.Delete)
		authorized.POST("/epg-sources/:id/trigger", epgCtrl.Trigger)
		authorized.GET("/epg-sources/:id/programs", epgCtrl.GetPrograms)
		authorized.GET("/epg-sources/:id/channels", epgCtrl.GetChannels)
		authorized.GET("/epg-sources/:id/dates", epgCtrl.GetDates)

		// Channel Logos
		logoCtrl := NewLogoController(logoDir)
		authorized.GET("/logos", logoCtrl.List)
		authorized.POST("/logos/upload", logoCtrl.Upload)
		authorized.POST("/logos/batch-upload", logoCtrl.BatchUpload)
		authorized.PUT("/logos/:id", logoCtrl.Update)
		authorized.DELETE("/logos/:id", logoCtrl.Delete)

		// Aggregation Rules
		ruleCtrl := NewRuleController()
		authorized.GET("/rules", ruleCtrl.List)
		authorized.GET("/rules/:id", ruleCtrl.Get)
		authorized.POST("/rules", ruleCtrl.Create)
		authorized.PUT("/rules/:id", ruleCtrl.Update)
		authorized.DELETE("/rules/:id", ruleCtrl.Delete)

		// Publish Interfaces
		publishCtrl := NewPublishController(scheduler)
		authorized.GET("/publish", publishCtrl.ListInterfaces)
		authorized.GET("/publish/:id", publishCtrl.GetInterface)
		authorized.POST("/publish", publishCtrl.CreateInterface)
		authorized.PUT("/publish/:id", publishCtrl.UpdateInterface)
		authorized.DELETE("/publish/:id", publishCtrl.DeleteInterface)
		authorized.POST("/publish/preview", publishCtrl.PreviewInterface)
	}

	// --- Embedded frontend (SPA with hash routing) ---
	// Read index.html once for SPA fallback
	indexHTML, _ := fs.ReadFile(frontendFS, "index.html")
	frontendHTTP := http.FileServer(http.FS(frontendFS))
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Don't serve frontend for API or other backend routes
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/sub/") || strings.HasPrefix(path, "/logo/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到该资源"})
			return
		}

		// Try to serve the requested static file (js, css, images, etc.)
		// Strip leading slash for fs.Open check
		filePath := strings.TrimPrefix(path, "/")
		if filePath != "" {
			if f, err := frontendFS.Open(filePath); err == nil {
				f.Close()
				frontendHTTP.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// File not found in embedded FS — serve index.html for SPA fallback
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	return r
}
