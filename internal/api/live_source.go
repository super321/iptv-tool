package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/internal/task"
	"iptv-tool-v2/pkg/utils"
)

// LiveSourceController handles CRUD operations for live sources
type LiveSourceController struct {
	liveService *service.LiveSourceService
	scheduler   *task.Scheduler
}

func NewLiveSourceController(scheduler *task.Scheduler) *LiveSourceController {
	return &LiveSourceController{
		liveService: service.NewLiveSourceService(),
		scheduler:   scheduler,
	}
}

// List returns all live sources with channel count
// GET /api/live-sources
func (lc *LiveSourceController) List(c *gin.Context) {
	var sources []model.LiveSource
	if err := model.DB.Order("id desc").Find(&sources).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type LiveSourceWithCount struct {
		model.LiveSource
		ChannelCount int64 `json:"channel_count"`
	}

	result := make([]LiveSourceWithCount, 0)
	for _, s := range sources {
		var channelCount int64
		model.DB.Model(&model.ParsedChannel{}).Where("source_id = ?", s.ID).Count(&channelCount)
		result = append(result, LiveSourceWithCount{
			LiveSource:   s,
			ChannelCount: channelCount,
		})
	}

	c.JSON(http.StatusOK, result)
}

// Get returns a single live source by ID
// GET /api/live-sources/:id
func (lc *LiveSourceController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var source model.LiveSource
	if err := model.DB.First(&source, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该直播源"})
		return
	}
	c.JSON(http.StatusOK, source)
}

// CreateLiveSourceRequest is the request body for creating a live source
type CreateLiveSourceRequest struct {
	Name        string               `json:"name" binding:"required"`
	Description string               `json:"description"`
	Type        model.LiveSourceType `json:"type" binding:"required"`
	URL         string               `json:"url"`
	Content     string               `json:"content"`
	Headers     json.RawMessage      `json:"headers"`
	CronTime    string               `json:"cron_time"`
	CronDetect  string               `json:"cron_detect"`
	IPTVConfig  json.RawMessage      `json:"iptv_config"`
	EPGEnabled  bool                 `json:"epg_enabled"` // Whether to auto-create EPG source
}

// Create adds a new live source
// POST /api/live-sources
func (lc *LiveSourceController) Create(c *gin.Context) {
	var req CreateLiveSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check name uniqueness
	var existing int64
	model.DB.Model(&model.LiveSource{}).Where("name = ?", req.Name).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该名称已存在，请换一个名称"})
		return
	}

	// Validate cron_time for non-manual sources
	if req.Type != model.LiveSourceTypeNetworkManual && req.CronTime != "" {
		if !task.ValidateCronTime(req.CronTime) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的定时刷新表达式"})
			return
		}
	}
	// Validate cron_detect
	if req.CronDetect != "" {
		if !task.ValidateCronTime(req.CronDetect) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的定时检测表达式"})
			return
		}
	}
	// network_manual sources must not have cron refresh (but can have cron detect)
	if req.Type == model.LiveSourceTypeNetworkManual {
		req.CronTime = ""
	}

	// Validate URL or content based on type
	var tvgURL string
	switch req.Type {
	case model.LiveSourceTypeNetworkURL:
		if req.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "网络链接类型需要提供URL"})
			return
		}
		var err error
		if _, tvgURL, err = lc.liveService.ValidateNetworkURL(req.URL, string(req.Headers)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	case model.LiveSourceTypeNetworkManual:
		if req.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "手动输入类型需要提供内容"})
			return
		}
		var err error
		if _, tvgURL, err = lc.liveService.ValidateManualContent(req.Content); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	case model.LiveSourceTypeIPTV:
		if len(req.IPTVConfig) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "IPTV类型需要提供IPTV配置"})
			return
		}
	}

	source := model.LiveSource{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		URL:         req.URL,
		Content:     req.Content,
		Headers:     string(req.Headers),
		CronTime:    req.CronTime,
		CronDetect:  req.CronDetect,
		Status:      true,
		IsSyncing:   true,
		IPTVConfig:  string(req.IPTVConfig),
	}

	if err := model.DB.Create(&source).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Auto-create EPG source if IPTV type with EPG enabled
	var createdEPGSource *model.EPGSource
	if req.Type == model.LiveSourceTypeIPTV && req.EPGEnabled {
		epgSource := model.EPGSource{
			Name:         source.Name + " - EPG",
			Type:         model.EPGSourceTypeIPTV,
			LiveSourceID: &source.ID,
			CronTime:     req.CronTime,
			Status:       true,
			IsSyncing:    true,
			IPTVConfig:   string(req.IPTVConfig),
		}
		if err := model.DB.Create(&epgSource).Error; err == nil {
			createdEPGSource = &epgSource
			// Schedule EPG source if cron is set
			if epgSource.CronTime != "" {
				lc.scheduler.AddEPGSourceTask(epgSource.ID, epgSource.CronTime)
			}
		}
	}

	// Auto-create EPG source from x-tvg-url for network_url/network_manual types
	if (req.Type == model.LiveSourceTypeNetworkURL || req.Type == model.LiveSourceTypeNetworkManual) && req.EPGEnabled && tvgURL != "" {
		// Check if an EPG source with the same URL already exists to avoid duplicates
		var existingCount int64
		model.DB.Model(&model.EPGSource{}).Where("url = ?", tvgURL).Count(&existingCount)
		if existingCount == 0 {
			epgSource := model.EPGSource{
				Name:      source.Name + " - EPG",
				Type:      model.EPGSourceTypeNetworkXMLTV,
				URL:       tvgURL,
				CronTime:  req.CronTime,
				Status:    true,
				IsSyncing: true,
			}
			if err := model.DB.Create(&epgSource).Error; err == nil {
				slog.Info("Auto-created EPG source from x-tvg-url", "epg_id", epgSource.ID, "url", tvgURL, "live_source", source.Name)
				if epgSource.CronTime != "" {
					lc.scheduler.AddEPGSourceTask(epgSource.ID, epgSource.CronTime)
				}
				lc.scheduler.TriggerEPGSourceNow(epgSource.ID)
			}
		} else {
			slog.Info("Skipped auto-create EPG source, URL already exists", "url", tvgURL)
		}
	}

	// Schedule cron task if applicable
	if source.CronTime != "" && source.Type != model.LiveSourceTypeNetworkManual {
		lc.scheduler.AddLiveSourceTask(source.ID, source.CronTime)
	}

	// Schedule detect cron task if applicable
	if source.CronDetect != "" {
		lc.scheduler.AddDetectTask(source.ID, source.CronDetect)
	}

	// Trigger initial fetch for Live Source
	lc.scheduler.TriggerLiveSourceNow(source.ID)

	// If an EPG source was auto-created, delay its initial fetch to prevent IPTV login collision
	if createdEPGSource != nil {
		go func(epgID uint) {
			// Delay for 30 seconds to allow LiveSource to finish its login & fetch
			time.Sleep(30 * time.Second)
			lc.scheduler.TriggerEPGSourceNow(epgID)
		}(createdEPGSource.ID)
	}

	c.JSON(http.StatusCreated, source)
}

// UpdateLiveSourceRequest is the request body for updating a live source
type UpdateLiveSourceRequest struct {
	Name          *string          `json:"name"`
	Description   *string          `json:"description"`
	URL           *string          `json:"url"`
	Content       *string          `json:"content"`
	Headers       *json.RawMessage `json:"headers"`
	CronTime      *string          `json:"cron_time"`
	CronDetect    *string          `json:"cron_detect"`
	Status        *bool            `json:"status"`
	IPTVConfig    *json.RawMessage `json:"iptv_config"`
	AutoCreateEPG *bool            `json:"auto_create_epg"` // Whether to auto-create EPG source from x-tvg-url
}

// Update modifies a live source
// PUT /api/live-sources/:id
func (lc *LiveSourceController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var source model.LiveSource
	if err := model.DB.First(&source, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该直播源"})
		return
	}

	var req UpdateLiveSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		// Check name uniqueness
		var existing int64
		model.DB.Model(&model.LiveSource{}).Where("name = ? AND id != ?", *req.Name, id).Count(&existing)
		if existing > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "该名称已存在，请换一个名称"})
			return
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Headers != nil {
		updates["headers"] = string(*req.Headers)
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.IPTVConfig != nil {
		updates["iptv_config"] = string(*req.IPTVConfig)
	}
	if req.CronTime != nil {
		if source.Type == model.LiveSourceTypeNetworkManual {
			updates["cron_time"] = "" // Force no cron for manual sources
		} else {
			if *req.CronTime != "" && !task.ValidateCronTime(*req.CronTime) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的定时刷新表达式"})
				return
			}
			updates["cron_time"] = *req.CronTime
		}
	}
	if req.CronDetect != nil {
		if *req.CronDetect != "" && !task.ValidateCronTime(*req.CronDetect) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的定时检测表达式"})
			return
		}
		updates["cron_detect"] = *req.CronDetect
	}

	if err := model.DB.Model(&source).Updates(updates).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload to get updated fields
	model.DB.First(&source, uint(id))

	// Auto-create EPG source from x-tvg-url if requested during update
	if req.AutoCreateEPG != nil && *req.AutoCreateEPG {
		if source.Type == model.LiveSourceTypeNetworkURL || source.Type == model.LiveSourceTypeNetworkManual {
			var tvgURL string
			if source.Type == model.LiveSourceTypeNetworkURL && source.URL != "" {
				_, tvgURL, _ = lc.liveService.ValidateNetworkURL(source.URL, source.Headers)
			} else if source.Type == model.LiveSourceTypeNetworkManual && source.Content != "" {
				_, tvgURL, _ = lc.liveService.ValidateManualContent(source.Content)
			}
			if tvgURL != "" {
				var existingCount int64
				model.DB.Model(&model.EPGSource{}).Where("url = ?", tvgURL).Count(&existingCount)
				if existingCount == 0 {
					epgSource := model.EPGSource{
						Name:      source.Name + " - EPG",
						Type:      model.EPGSourceTypeNetworkXMLTV,
						URL:       tvgURL,
						CronTime:  source.CronTime,
						Status:    true,
						IsSyncing: true,
					}
					if err := model.DB.Create(&epgSource).Error; err == nil {
						slog.Info("Auto-created EPG source from x-tvg-url (update)", "epg_id", epgSource.ID, "url", tvgURL, "live_source", source.Name)
						if epgSource.CronTime != "" {
							lc.scheduler.AddEPGSourceTask(epgSource.ID, epgSource.CronTime)
						}
						lc.scheduler.TriggerEPGSourceNow(epgSource.ID)
					}
				} else {
					slog.Info("Skipped auto-create EPG source (update), URL already exists", "url", tvgURL)
				}
			}
		}
	}

	// Update scheduler for refresh tasks
	if source.Type != model.LiveSourceTypeNetworkManual && source.CronTime != "" && source.Status {
		lc.scheduler.AddLiveSourceTask(source.ID, source.CronTime)
	} else {
		lc.scheduler.RemoveLiveSourceTask(source.ID)
	}

	// Update scheduler for detect tasks
	if source.CronDetect != "" && source.Status {
		lc.scheduler.AddDetectTask(source.ID, source.CronDetect)
	} else {
		lc.scheduler.RemoveDetectTask(source.ID)
	}

	c.JSON(http.StatusOK, source)
}

// Delete removes a live source
// DELETE /api/live-sources/:id
func (lc *LiveSourceController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	sourceID := uint(id)

	// === Phase 1: Defensive Checks (Read-Only) ===

	// 1. Check if this source is referenced by any live publish interface
	var publishInterfaces []model.PublishInterface
	model.DB.Where("type = ?", "live").Find(&publishInterfaces)
	for _, pi := range publishInterfaces {
		for _, idStr := range strings.Split(pi.SourceIDs, ",") {
			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				continue
			}
			refID, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				continue
			}
			if uint(refID) == sourceID {
				c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("该直播源正被发布接口「%s」引用，请先移除引用后再删除", pi.Name)})
				return
			}
		}
	}

	// 2. Fetch associated EPG sources (auto-created ones)
	var epgSources []model.EPGSource
	model.DB.Where("live_source_id = ?", sourceID).Find(&epgSources)

	// 3. Check if any of these auto-created EPG sources are referenced by any EPG publish interface
	if len(epgSources) > 0 {
		var epgPublishInterfaces []model.PublishInterface
		model.DB.Where("type = ?", "epg").Find(&epgPublishInterfaces)

		for _, epg := range epgSources {
			epgIDStr := strconv.FormatUint(uint64(epg.ID), 10)
			for _, pi := range epgPublishInterfaces {
				if pi.SourceIDs == "" {
					continue
				}
				for _, rID := range strings.Split(pi.SourceIDs, ",") {
					if strings.TrimSpace(rID) == epgIDStr {
						c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("该直播源关联的EPG源正被发布接口「%s」引用，请先移除EPG接口的引用后再删除直播源", pi.Name)})
						return
					}
				}
			}
		}
	}

	// === Phase 2: Cascading Deletion (Execute safely only after all checks pass) ===

	// Remove from scheduler
	lc.scheduler.RemoveLiveSourceTask(sourceID)
	lc.scheduler.RemoveDetectTask(sourceID)

	// Delete associated parsed channels
	model.DB.Where("source_id = ?", sourceID).Delete(&model.ParsedChannel{})

	// Delete associated EPG sources and their scheduler tasks
	for _, epg := range epgSources {
		lc.scheduler.RemoveEPGSourceTask(epg.ID)
		model.DB.Where("source_id = ?", epg.ID).Delete(&model.ParsedEPG{})
	}
	model.DB.Where("live_source_id = ?", sourceID).Delete(&model.EPGSource{})

	// Delete the live source itself
	if err := model.DB.Delete(&model.LiveSource{}, sourceID).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "直播源已删除"})
}

// Trigger manually triggers a fetch for a live source
// POST /api/live-sources/:id/trigger
func (lc *LiveSourceController) Trigger(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	model.DB.Model(&model.LiveSource{}).Where("id = ?", uint(id)).Update("is_syncing", true)

	lc.scheduler.TriggerLiveSourceNow(uint(id))
	c.JSON(http.StatusOK, gin.H{"message": "已触发抓取"})
}

// GetChannels returns parsed channels for a live source
// GET /api/live-sources/:id/channels
func (lc *LiveSourceController) GetChannels(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var channels []model.ParsedChannel
	if err := model.DB.Where("source_id = ?", uint(id)).Find(&channels).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sort.Slice(channels, func(i, j int) bool {
		if channels[i].Name == channels[j].Name {
			return utils.NaturalLess(channels[i].TVGId, channels[j].TVGId)
		}
		return utils.NaturalLess(channels[i].Name, channels[j].Name)
	})
	c.JSON(http.StatusOK, gin.H{
		"total":    len(channels),
		"channels": channels,
	})
}

// TriggerDetect manually triggers channel detection for a live source
// POST /api/live-sources/:id/detect
func (lc *LiveSourceController) TriggerDetect(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	// Check if ffprobe is available before triggering detection
	if err := lc.scheduler.CheckFFprobe(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先在系统设置中上传 ffprobe 文件"})
		return
	}

	lc.scheduler.TriggerDetectNow(uint(id))
	c.JSON(http.StatusOK, gin.H{"message": "已触发检测"})
}

// UnlinkedIPTV returns IPTV live sources that do NOT have an associated EPG source
// GET /api/live-sources/unlinked-iptv
func (lc *LiveSourceController) UnlinkedIPTV(c *gin.Context) {
	var sources []model.LiveSource
	// Find IPTV live sources where no EPG source has live_source_id pointing to them
	if err := model.DB.Where("type = ? AND id NOT IN (?)",
		model.LiveSourceTypeIPTV,
		model.DB.Model(&model.EPGSource{}).Select("live_source_id").Where("live_source_id IS NOT NULL"),
	).Find(&sources).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sources)
}
