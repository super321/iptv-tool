package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/internal/task"
	"iptv-tool-v2/pkg/i18n"
)

// GeoIPController handles GeoIP database management
type GeoIPController struct {
	geoipSvc  *service.GeoIPService
	scheduler *task.Scheduler
}

func NewGeoIPController(geoipSvc *service.GeoIPService, scheduler *task.Scheduler) *GeoIPController {
	return &GeoIPController{geoipSvc: geoipSvc, scheduler: scheduler}
}

// GeoIPStatusResponse is the response for GET geoip status
type GeoIPStatusResponse struct {
	Exists         bool                 `json:"exists"`
	Version        string               `json:"version"`
	AutoUpdate     bool                 `json:"auto_update"`
	ScheduleConfig *task.ScheduleConfig `json:"schedule_config,omitempty"`
}

// GetStatus returns the current GeoIP database status
// GET /api/settings/geoip/status
func (gc *GeoIPController) GetStatus(c *gin.Context) {
	exists := gc.geoipSvc.DBExists()
	version := ""
	if exists {
		version = gc.geoipSvc.GetVersion()
	}
	autoUpdate, scheduleCfg := gc.geoipSvc.GetAutoUpdateConfig()

	c.JSON(http.StatusOK, GeoIPStatusResponse{
		Exists:         exists,
		Version:        version,
		AutoUpdate:     autoUpdate,
		ScheduleConfig: &scheduleCfg,
	})
}

// CheckUpdate starts downloading the latest GeoIP database asynchronously.
// POST /api/settings/geoip/check-update
func (gc *GeoIPController) CheckUpdate(c *gin.Context) {
	lang := i18n.Lang(c)

	// Check if already downloading
	if gc.geoipSvc.IsDownloading() {
		c.JSON(http.StatusConflict, gin.H{
			"error": i18n.T(lang, "error.geoip_downloading"),
		})
		return
	}

	// Start download in background
	go func() {
		if err := gc.geoipSvc.DownloadAndExtract(); err != nil {
			slog.Error("GeoIP download failed", "error", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "message.geoip_download_started"),
	})
}

// GetDownloadProgress returns the current download progress
// GET /api/settings/geoip/progress
func (gc *GeoIPController) GetDownloadProgress(c *gin.Context) {
	progress := gc.geoipSvc.GetDownloadProgress()

	// If download just completed, also return the new version
	version := ""
	if !progress.Downloading && gc.geoipSvc.DBExists() {
		version = gc.geoipSvc.GetVersion()
	}

	c.JSON(http.StatusOK, gin.H{
		"downloading":      progress.Downloading,
		"downloaded_bytes": progress.DownloadedBytes,
		"total_bytes":      progress.TotalBytes,
		"percent":          progress.Percent,
		"attempt":          progress.Attempt,
		"max_retries":      progress.MaxRetries,
		"error":            progress.Error,
		"version":          version,
		"exists":           gc.geoipSvc.DBExists(),
	})
}

// UpdateAutoUpdateRequest is the request for updating geoip auto-update settings
type UpdateAutoUpdateRequest struct {
	Enabled        bool                 `json:"enabled"`
	ScheduleConfig *task.ScheduleConfig `json:"schedule_config"`
}

// UpdateAutoUpdate saves the auto-update settings and manages the scheduler task
// PUT /api/settings/geoip/auto-update
func (gc *GeoIPController) UpdateAutoUpdate(c *gin.Context) {
	var req UpdateAutoUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate schedule config if enabled
	if req.Enabled && req.ScheduleConfig != nil {
		if err := task.ValidateScheduleConfig(req.ScheduleConfig, i18n.Lang(c)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), err.Error())})
			return
		}
	}

	// Save auto-update with schedule config (only update config if provided)
	if req.ScheduleConfig != nil {
		gc.geoipSvc.SaveAutoUpdateConfig(req.Enabled, *req.ScheduleConfig)
	} else {
		// Only update enabled flag, keep existing schedule config
		_, existingCfg := gc.geoipSvc.GetAutoUpdateConfig()
		gc.geoipSvc.SaveAutoUpdateConfig(req.Enabled, existingCfg)
	}

	// Get the actual saved config to return and use for scheduler
	_, savedCfg := gc.geoipSvc.GetAutoUpdateConfig()

	// Manage scheduler task
	if req.Enabled {
		gc.scheduler.AddGeoIPUpdateTask(&savedCfg)
	} else {
		gc.scheduler.RemoveGeoIPUpdateTask()
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         i18n.T(i18n.Lang(c), "message.geoip_auto_update_saved"),
		"enabled":         req.Enabled,
		"schedule_config": savedCfg,
	})
}
