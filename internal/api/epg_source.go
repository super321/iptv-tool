package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/publish"
	"iptv-tool-v2/internal/task"
	"iptv-tool-v2/pkg/i18n"
	"iptv-tool-v2/pkg/utils"
)

// EPGSourceController handles CRUD operations for EPG sources
type EPGSourceController struct {
	scheduler *task.Scheduler
}

func NewEPGSourceController(scheduler *task.Scheduler) *EPGSourceController {
	return &EPGSourceController{
		scheduler: scheduler,
	}
}

// List returns all EPG sources with channel/program counts
// GET /api/epg-sources
func (ec *EPGSourceController) List(c *gin.Context) {
	var sources []model.EPGSource
	if err := model.DB.Order("id desc").Find(&sources).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build response with counts
	type EPGSourceWithCounts struct {
		model.EPGSource
		ChannelCount int64 `json:"channel_count"`
		ProgramCount int64 `json:"program_count"`
	}

	result := make([]EPGSourceWithCounts, 0)
	for _, s := range sources {
		var channelCount int64
		var programCount int64
		model.DB.Model(&model.ParsedEPG{}).Where("source_id = ?", s.ID).
			Select("COUNT(DISTINCT channel)").Scan(&channelCount)
		model.DB.Model(&model.ParsedEPG{}).Where("source_id = ?", s.ID).Count(&programCount)
		result = append(result, EPGSourceWithCounts{
			EPGSource:    s,
			ChannelCount: channelCount,
			ProgramCount: programCount,
		})
	}

	c.JSON(http.StatusOK, result)
}

// Get returns a single EPG source by ID
// GET /api/epg-sources/:id
func (ec *EPGSourceController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	var source model.EPGSource
	if err := model.DB.First(&source, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.epg_source_not_found")})
		return
	}
	c.JSON(http.StatusOK, source)
}

// CreateEPGSourceRequest is the request body for creating an EPG source
type CreateEPGSourceRequest struct {
	Name         string              `json:"name" binding:"required"`
	Description  string              `json:"description"`
	Type         model.EPGSourceType `json:"type" binding:"required"`
	URL          string              `json:"url"`
	CronTime     string              `json:"cron_time"`
	LiveSourceID *uint               `json:"live_source_id"` // For IPTV type: link to an existing IPTV live source
	EPGStrategy  string              `json:"epg_strategy"`   // For IPTV type: strategy name (auto, vsp, etc.)
	IPTVConfig   json.RawMessage     `json:"iptv_config"`
}

// Create adds a new EPG source
// POST /api/epg-sources
func (ec *EPGSourceController) Create(c *gin.Context) {
	var req CreateEPGSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Trim whitespace from string inputs
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.URL = strings.TrimSpace(req.URL)
	req.CronTime = strings.TrimSpace(req.CronTime)

	if req.CronTime != "" && !task.ValidateInterval(req.CronTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_refresh_interval")})
		return
	}

	var liveSourceID *uint
	var iptvConfigStr string

	switch req.Type {
	case model.EPGSourceTypeNetworkXMLTV:
		if req.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.xmltv_url_required")})
			return
		}
	case model.EPGSourceTypeIPTV:
		if req.LiveSourceID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.iptv_source_id_required")})
			return
		}
		// Verify the live source exists and is IPTV type
		var liveSource model.LiveSource
		if err := model.DB.First(&liveSource, *req.LiveSourceID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.linked_source_not_found")})
			return
		}
		if liveSource.Type != model.LiveSourceTypeIPTV {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.linked_source_not_iptv")})
			return
		}
		// Check it's not already linked to another EPG source
		var existingCount int64
		model.DB.Model(&model.EPGSource{}).Where("live_source_id = ?", *req.LiveSourceID).Count(&existingCount)
		if existingCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.source_already_linked")})
			return
		}
		liveSourceID = req.LiveSourceID

		// Build iptv_config from live source's config + epg_strategy override
		iptvConfig := liveSource.IPTVConfig
		if req.EPGStrategy != "" {
			// Merge strategy into the config JSON
			var configMap map[string]interface{}
			if err := json.Unmarshal([]byte(iptvConfig), &configMap); err != nil {
				configMap = make(map[string]interface{})
			}
			configMap["epgStrategy"] = req.EPGStrategy
			merged, _ := json.Marshal(configMap)
			iptvConfig = string(merged)
		}
		iptvConfigStr = iptvConfig
	}

	// If raw iptv_config was provided directly (legacy path), use it
	if len(req.IPTVConfig) > 0 && iptvConfigStr == "" {
		iptvConfigStr = string(req.IPTVConfig)
	}

	source := model.EPGSource{
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		URL:          req.URL,
		CronTime:     req.CronTime,
		LiveSourceID: liveSourceID,
		Status:       true,
		IsSyncing:    true,
		IPTVConfig:   iptvConfigStr,
	}

	if err := model.DB.Create(&source).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Schedule refresh task if applicable
	if source.CronTime != "" {
		ec.scheduler.AddEPGSourceTask(source.ID, source.CronTime)
	}

	// Trigger initial fetch
	ec.scheduler.TriggerEPGSourceNow(source.ID)

	publish.InvalidateAll()
	c.JSON(http.StatusCreated, source)
}

// UpdateEPGSourceRequest is the request body for updating an EPG source
type UpdateEPGSourceRequest struct {
	Name        *string          `json:"name"`
	Description *string          `json:"description"`
	URL         *string          `json:"url"`
	CronTime    *string          `json:"cron_time"`
	Status      *bool            `json:"status"`
	EPGStrategy *string          `json:"epg_strategy"`
	IPTVConfig  *json.RawMessage `json:"iptv_config"`
}

// Update modifies an EPG source
// PUT /api/epg-sources/:id
func (ec *EPGSourceController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	var source model.EPGSource
	if err := model.DB.First(&source, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.epg_source_not_found")})
		return
	}

	var req UpdateEPGSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Trim whitespace from string inputs
	if req.Name != nil {
		*req.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		*req.Description = strings.TrimSpace(*req.Description)
	}
	if req.URL != nil {
		*req.URL = strings.TrimSpace(*req.URL)
	}
	if req.CronTime != nil {
		*req.CronTime = strings.TrimSpace(*req.CronTime)
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.IPTVConfig != nil {
		updates["iptv_config"] = string(*req.IPTVConfig)
	}
	// Handle EPG strategy update by merging into existing iptv_config
	if req.EPGStrategy != nil && source.Type == model.EPGSourceTypeIPTV {
		configStr := source.IPTVConfig
		if raw, ok := updates["iptv_config"]; ok {
			configStr = raw.(string)
		}
		var configMap map[string]interface{}
		if err := json.Unmarshal([]byte(configStr), &configMap); err != nil {
			configMap = make(map[string]interface{})
		}
		configMap["epgStrategy"] = *req.EPGStrategy
		merged, _ := json.Marshal(configMap)
		updates["iptv_config"] = string(merged)
	}
	if req.CronTime != nil {
		if *req.CronTime != "" && !task.ValidateInterval(*req.CronTime) {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_refresh_interval")})
			return
		}
		updates["cron_time"] = *req.CronTime
	}

	if err := model.DB.Model(&source).Updates(updates).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	model.DB.First(&source, uint(id))

	// Update scheduler
	if source.CronTime != "" && source.Status {
		ec.scheduler.AddEPGSourceTask(source.ID, source.CronTime)
	} else {
		ec.scheduler.RemoveEPGSourceTask(source.ID)
	}

	publish.InvalidateAll()
	c.JSON(http.StatusOK, source)
}

// Delete removes an EPG source
// DELETE /api/epg-sources/:id
func (ec *EPGSourceController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	sourceID := uint(id)

	// Check if this source is referenced by any publish interface
	var publishInterfaces []model.PublishInterface
	model.DB.Where("type = ?", "epg").Find(&publishInterfaces)
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
				c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.epg_ref_publish", pi.Name)})
				return
			}
		}
	}

	// Remove from scheduler
	ec.scheduler.RemoveEPGSourceTask(sourceID)

	// Delete associated parsed EPG data
	model.DB.Where("source_id = ?", sourceID).Delete(&model.ParsedEPG{})

	if err := model.DB.Delete(&model.EPGSource{}, sourceID).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	publish.InvalidateAll()
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(i18n.Lang(c), "message.epg_source_deleted")})
}

// Trigger manually triggers a fetch for an EPG source
// POST /api/epg-sources/:id/trigger
func (ec *EPGSourceController) Trigger(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	model.DB.Model(&model.EPGSource{}).Where("id = ?", uint(id)).Update("is_syncing", true)

	ec.scheduler.TriggerEPGSourceNow(uint(id))
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(i18n.Lang(c), "message.trigger_fetch")})
}

// GetPrograms returns parsed EPG programs for an EPG source
// GET /api/epg-sources/:id/programs
// Supports query params: ?channel=xxx&date=2025-01-01
func (ec *EPGSourceController) GetPrograms(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	// Support optional channel filter
	channelFilter := c.Query("channel")
	dateFilter := c.Query("date")

	query := model.DB.Where("source_id = ?", uint(id))
	if channelFilter != "" {
		query = query.Where("channel = ?", channelFilter)
	}
	if dateFilter != "" {
		// Filter by date: start_time between date 00:00:00 and date+1 00:00:00
		query = query.Where("DATE(start_time, 'localtime') = ?", dateFilter)
	}

	var programs []model.ParsedEPG
	if err := query.Order("start_time asc").Limit(1000).Find(&programs).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Count total
	var total int64
	countQuery := model.DB.Model(&model.ParsedEPG{}).Where("source_id = ?", uint(id))
	if channelFilter != "" {
		countQuery = countQuery.Where("channel = ?", channelFilter)
	}
	if dateFilter != "" {
		countQuery = countQuery.Where("DATE(start_time, 'localtime') = ?", dateFilter)
	}
	countQuery.Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"total":    total,
		"programs": programs,
	})
}

// GetChannels returns distinct channel list for an EPG source (Level 1 drill-down)
// GET /api/epg-sources/:id/channels
func (ec *EPGSourceController) GetChannels(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	type ChannelInfo struct {
		Channel     string `json:"channel"`
		ChannelName string `json:"channel_name"`
		Count       int64  `json:"count"`
	}

	var channels []ChannelInfo
	if err := model.DB.Model(&model.ParsedEPG{}).
		Select("channel, MAX(channel_name) as channel_name, COUNT(*) as count").
		Where("source_id = ?", uint(id)).
		Group("channel").
		Find(&channels).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sort.Slice(channels, func(i, j int) bool {
		if channels[i].ChannelName == channels[j].ChannelName {
			return utils.NaturalLess(channels[i].Channel, channels[j].Channel)
		}
		return utils.NaturalLess(channels[i].ChannelName, channels[j].ChannelName)
	})

	c.JSON(http.StatusOK, gin.H{
		"total":    len(channels),
		"channels": channels,
	})
}

// GetDates returns distinct dates for a specific channel in an EPG source (Level 2 drill-down)
// GET /api/epg-sources/:id/dates?channel=xxx
func (ec *EPGSourceController) GetDates(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	channelFilter := c.Query("channel")
	if channelFilter == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.missing_channel_param")})
		return
	}

	type DateInfo struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	var dates []DateInfo
	if err := model.DB.Model(&model.ParsedEPG{}).
		Select("DATE(start_time, 'localtime') as date, COUNT(*) as count").
		Where("source_id = ? AND channel = ?", uint(id), channelFilter).
		Group("DATE(start_time, 'localtime')").
		Order("date asc").
		Find(&dates).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(dates),
		"dates": dates,
	})
}
