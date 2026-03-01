package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/internal/task"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type LiveSourceWithCount struct {
		model.LiveSource
		ChannelCount int64 `json:"channel_count"`
	}

	var result []LiveSourceWithCount
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var source model.LiveSource
	if err := model.DB.First(&source, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "live source not found"})
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
	CronTime    string               `json:"cron_time"`
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron_time value"})
			return
		}
	}
	// network_manual sources must not have cron
	if req.Type == model.LiveSourceTypeNetworkManual {
		req.CronTime = ""
	}

	// Validate URL or content based on type
	switch req.Type {
	case model.LiveSourceTypeNetworkURL:
		if req.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url is required for network_url type"})
			return
		}
		if _, err := lc.liveService.ValidateNetworkURL(req.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	case model.LiveSourceTypeNetworkManual:
		if req.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content is required for network_manual type"})
			return
		}
		if _, err := lc.liveService.ValidateManualContent(req.Content); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	case model.LiveSourceTypeIPTV:
		if len(req.IPTVConfig) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "iptv_config is required for iptv type"})
			return
		}
	}

	source := model.LiveSource{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		URL:         req.URL,
		Content:     req.Content,
		CronTime:    req.CronTime,
		Status:      true,
		IPTVConfig:  string(req.IPTVConfig),
	}

	if err := model.DB.Create(&source).Error; err != nil {
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

	// Schedule cron task if applicable
	if source.CronTime != "" && source.Type != model.LiveSourceTypeNetworkManual {
		lc.scheduler.AddLiveSourceTask(source.ID, source.CronTime)
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
	Name        *string          `json:"name"`
	Description *string          `json:"description"`
	URL         *string          `json:"url"`
	Content     *string          `json:"content"`
	CronTime    *string          `json:"cron_time"`
	Status      *bool            `json:"status"`
	IPTVConfig  *json.RawMessage `json:"iptv_config"`
}

// Update modifies a live source
// PUT /api/live-sources/:id
func (lc *LiveSourceController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var source model.LiveSource
	if err := model.DB.First(&source, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "live source not found"})
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
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron_time value"})
				return
			}
			updates["cron_time"] = *req.CronTime
		}
	}

	if err := model.DB.Model(&source).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload to get updated fields
	model.DB.First(&source, uint(id))

	// Update scheduler
	if source.Type != model.LiveSourceTypeNetworkManual && source.CronTime != "" && source.Status {
		lc.scheduler.AddLiveSourceTask(source.ID, source.CronTime)
	} else {
		lc.scheduler.RemoveLiveSourceTask(source.ID)
	}

	c.JSON(http.StatusOK, source)
}

// Delete removes a live source
// DELETE /api/live-sources/:id
func (lc *LiveSourceController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "live source deleted"})
}

// Trigger manually triggers a fetch for a live source
// POST /api/live-sources/:id/trigger
func (lc *LiveSourceController) Trigger(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	lc.scheduler.TriggerLiveSourceNow(uint(id))
	c.JSON(http.StatusOK, gin.H{"message": "fetch triggered"})
}

// GetChannels returns parsed channels for a live source
// GET /api/live-sources/:id/channels
func (lc *LiveSourceController) GetChannels(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var channels []model.ParsedChannel
	if err := model.DB.Where("source_id = ?", uint(id)).Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total":    len(channels),
		"channels": channels,
	})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sources)
}
