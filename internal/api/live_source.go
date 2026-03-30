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
	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/internal/task"
	"iptv-tool-v2/pkg/i18n"
	"iptv-tool-v2/pkg/utils"
)

// translateError checks if the error message is an i18n key (starts with "error.")
// and translates it. Otherwise, returns the error message as-is.
func translateError(lang string, err error) string {
	msg := err.Error()
	if strings.HasPrefix(msg, "error.") {
		return i18n.T(lang, msg)
	}
	return msg
}

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

	// Single GROUP BY query to get all channel counts at once (replaces N+1 queries)
	type sourceCount struct {
		SourceID uint  `gorm:"column:source_id"`
		Count    int64 `gorm:"column:count"`
	}
	var counts []sourceCount
	model.DB.Model(&model.ParsedChannel{}).
		Select("source_id, COUNT(*) as count").
		Group("source_id").Find(&counts)

	countMap := make(map[uint]int64, len(counts))
	for _, sc := range counts {
		countMap[sc.SourceID] = sc.Count
	}

	type LiveSourceWithCount struct {
		model.LiveSource
		ChannelCount int64 `json:"channel_count"`
	}

	result := make([]LiveSourceWithCount, 0, len(sources))
	for _, s := range sources {
		result = append(result, LiveSourceWithCount{
			LiveSource:   s,
			ChannelCount: countMap[s.ID],
		})
	}

	c.JSON(http.StatusOK, result)
}

// Get returns a single live source by ID
// GET /api/live-sources/:id
func (lc *LiveSourceController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	var source model.LiveSource
	if err := model.DB.First(&source, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.live_source_not_found")})
		return
	}
	c.JSON(http.StatusOK, source)
}

// CreateLiveSourceRequest is the request body for creating a live source
type CreateLiveSourceRequest struct {
	Name           string               `json:"name" binding:"required"`
	Description    string               `json:"description"`
	Type           model.LiveSourceType `json:"type" binding:"required"`
	URL            string               `json:"url"`
	Content        string               `json:"content"`
	Headers        json.RawMessage      `json:"headers"`
	CronTime       string               `json:"cron_time"`
	CronDetect     string               `json:"cron_detect"`
	DetectStrategy string               `json:"detect_strategy"`
	IPTVConfig     json.RawMessage      `json:"iptv_config"`
	EPGEnabled     bool                 `json:"epg_enabled"` // Whether to auto-create EPG source
}

// Create adds a new live source
// POST /api/live-sources
func (lc *LiveSourceController) Create(c *gin.Context) {
	var req CreateLiveSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Trim whitespace from string inputs
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.URL = strings.TrimSpace(req.URL)
	req.CronTime = strings.TrimSpace(req.CronTime)
	req.CronDetect = strings.TrimSpace(req.CronDetect)

	// Check name uniqueness
	var existing int64
	model.DB.Model(&model.LiveSource{}).Where("name = ?", req.Name).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.name_exists")})
		return
	}

	// Validate refresh schedule for non-manual sources
	if req.Type != model.LiveSourceTypeNetworkManual && req.CronTime != "" {
		cronCfg, err := task.ParseScheduleConfig(req.CronTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_refresh_interval")})
			return
		}
		if err := task.ValidateScheduleConfig(cronCfg, i18n.Lang(c)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), err.Error())})
			return
		}
	}
	// Validate detect schedule
	if req.CronDetect != "" {
		detectCfg, err := task.ParseScheduleConfig(req.CronDetect)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_detect_interval")})
			return
		}
		if err := task.ValidateScheduleConfig(detectCfg, i18n.Lang(c)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), err.Error())})
			return
		}
	}
	// network_manual sources must not have scheduled refresh (but can have scheduled detect)
	if req.Type == model.LiveSourceTypeNetworkManual {
		req.CronTime = ""
	}

	// Validate URL or content based on type
	var tvgURL string
	switch req.Type {
	case model.LiveSourceTypeNetworkURL:
		if req.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.url_required_network")})
			return
		}
		var err error
		if _, tvgURL, err = lc.liveService.ValidateNetworkURL(req.URL, string(req.Headers)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": translateError(i18n.Lang(c), err)})
			return
		}
	case model.LiveSourceTypeNetworkManual:
		if req.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.content_required_manual")})
			return
		}
		var err error
		if _, tvgURL, err = lc.liveService.ValidateManualContent(req.Content); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": translateError(i18n.Lang(c), err)})
			return
		}
	case model.LiveSourceTypeIPTV:
		if len(req.IPTVConfig) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.iptv_config_required")})
			return
		}
	}

	source := model.LiveSource{
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		URL:            req.URL,
		Content:        req.Content,
		Headers:        string(req.Headers),
		CronTime:       req.CronTime,
		CronDetect:     req.CronDetect,
		DetectStrategy: req.DetectStrategy,
		Status:         true,
		IsSyncing:      true,
		IPTVConfig:     string(req.IPTVConfig),
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
			// Schedule EPG source if interval is set
			if epgSource.CronTime != "" {
				if epgCfg, err := task.ParseScheduleConfig(epgSource.CronTime); err == nil {
					lc.scheduler.AddEPGSourceTask(epgSource.ID, epgCfg)
				}
			}
		}
	}

	// Auto-create EPG source from x-tvg-url for network_url/network_manual types
	var epgSkippedWarning string
	if (req.Type == model.LiveSourceTypeNetworkURL || req.Type == model.LiveSourceTypeNetworkManual) && req.EPGEnabled && tvgURL != "" {
		// Check if an EPG source with the same URL already exists to avoid duplicates
		var existingCount int64
		model.DB.Model(&model.EPGSource{}).Where("url = ?", tvgURL).Count(&existingCount)
		if existingCount == 0 {
			epgSource := model.EPGSource{
				Name:      source.Name + " - EPG",
				Type:      model.EPGSourceTypeNetworkXMLTV,
				URL:       tvgURL,
				Headers:   source.Headers,
				CronTime:  req.CronTime,
				Status:    true,
				IsSyncing: true,
			}
			if err := model.DB.Create(&epgSource).Error; err == nil {
				slog.Info("Auto-created EPG source from x-tvg-url", "epg_id", epgSource.ID, "url", tvgURL, "live_source", source.Name)
				if epgSource.CronTime != "" {
					if epgCfg, err := task.ParseScheduleConfig(epgSource.CronTime); err == nil {
						lc.scheduler.AddEPGSourceTask(epgSource.ID, epgCfg)
					}
				}
				lc.scheduler.TriggerEPGSourceNow(epgSource.ID)
			}
		} else {
			slog.Info("Skipped auto-create EPG source, URL already exists", "url", tvgURL)
			epgSkippedWarning = i18n.T(i18n.Lang(c), "message.epg_url_exists")
		}
	}

	// Schedule refresh task if applicable
	if source.CronTime != "" && source.Type != model.LiveSourceTypeNetworkManual {
		if cfg, err := task.ParseScheduleConfig(source.CronTime); err == nil {
			lc.scheduler.AddLiveSourceTask(source.ID, cfg)
		}
	}

	// Schedule detect task if applicable
	if source.CronDetect != "" {
		if cfg, err := task.ParseScheduleConfig(source.CronDetect); err == nil {
			lc.scheduler.AddDetectTask(source.ID, cfg, source.DetectStrategy)
		}
	}

	// Trigger initial fetch for Live Source
	lc.scheduler.TriggerLiveSourceNow(source.ID)

	// If an EPG source was auto-created, trigger its initial fetch
	// The IPTV mutex ensures it will wait for the live source fetch to complete
	if createdEPGSource != nil {
		lc.scheduler.TriggerEPGSourceNow(createdEPGSource.ID)
	}

	publish.InvalidateAll()

	if epgSkippedWarning != "" {
		c.JSON(http.StatusCreated, gin.H{"data": source, "warning": epgSkippedWarning})
	} else {
		c.JSON(http.StatusCreated, source)
	}
}

// UpdateLiveSourceRequest is the request body for updating a live source
type UpdateLiveSourceRequest struct {
	Name           *string          `json:"name"`
	Description    *string          `json:"description"`
	URL            *string          `json:"url"`
	Content        *string          `json:"content"`
	Headers        *json.RawMessage `json:"headers"`
	CronTime       *string          `json:"cron_time"`
	CronDetect     *string          `json:"cron_detect"`
	DetectStrategy *string          `json:"detect_strategy"`
	Status         *bool            `json:"status"`
	IPTVConfig     *json.RawMessage `json:"iptv_config"`
	AutoCreateEPG  *bool            `json:"auto_create_epg"` // Whether to auto-create EPG source from x-tvg-url
}

// Update modifies a live source
// PUT /api/live-sources/:id
func (lc *LiveSourceController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	var source model.LiveSource
	if err := model.DB.First(&source, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.live_source_not_found")})
		return
	}

	var req UpdateLiveSourceRequest
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
	if req.CronDetect != nil {
		*req.CronDetect = strings.TrimSpace(*req.CronDetect)
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		// Check name uniqueness
		var existing int64
		model.DB.Model(&model.LiveSource{}).Where("name = ? AND id != ?", *req.Name, id).Count(&existing)
		if existing > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.name_exists")})
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
			updates["cron_time"] = "" // Force no refresh schedule for manual sources
		} else {
			if *req.CronTime != "" {
				cronCfg, err := task.ParseScheduleConfig(*req.CronTime)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_refresh_interval")})
					return
				}
				if err := task.ValidateScheduleConfig(cronCfg, i18n.Lang(c)); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), err.Error())})
					return
				}
			}
			updates["cron_time"] = *req.CronTime
		}
	}
	if req.CronDetect != nil {
		if *req.CronDetect != "" {
			detectCfg, err := task.ParseScheduleConfig(*req.CronDetect)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_detect_interval")})
				return
			}
			if err := task.ValidateScheduleConfig(detectCfg, i18n.Lang(c)); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), err.Error())})
				return
			}
		}
		updates["cron_detect"] = *req.CronDetect
	}
	if req.DetectStrategy != nil {
		updates["detect_strategy"] = *req.DetectStrategy
	}

	if err := model.DB.Model(&source).Updates(updates).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload to get updated fields
	model.DB.First(&source, uint(id))

	// Auto-update linked IPTV EPG source if LiveSource is IPTV type and IPTVConfig is updated
	// Only sync IPTV connection parameters, preserving EPG-specific fields (e.g. epgStrategy)
	if source.Type == model.LiveSourceTypeIPTV && req.IPTVConfig != nil {
		var linkedEPGs []model.EPGSource
		if err := model.DB.Where("live_source_id = ? AND type = ?", source.ID, model.EPGSourceTypeIPTV).
			Find(&linkedEPGs).Error; err != nil {
			slog.Error("Failed to find linked EPG sources", "error", err, "live_source_id", source.ID)
		} else {
			for _, epg := range linkedEPGs {
				// Parse the new live source config
				var newConfig map[string]interface{}
				if err := json.Unmarshal(*req.IPTVConfig, &newConfig); err != nil {
					slog.Error("Failed to parse new IPTV config", "error", err)
					continue
				}

				// Preserve the EPG source's existing epgStrategy
				var oldConfig map[string]interface{}
				if err := json.Unmarshal([]byte(epg.IPTVConfig), &oldConfig); err == nil {
					if strategy, ok := oldConfig["epgStrategy"]; ok {
						newConfig["epgStrategy"] = strategy
					}
				}

				merged, _ := json.Marshal(newConfig)
				if err := model.DB.Model(&epg).Update("iptv_config", string(merged)).Error; err != nil {
					slog.Error("Failed to auto-update linked EPG source IPTV config", "error", err, "epg_id", epg.ID)
				} else {
					slog.Info("Auto-updated linked EPG source IPTV config", "live_source_id", source.ID, "epg_id", epg.ID)
				}
			}
		}
	}

	// Auto-create EPG source from x-tvg-url if requested during update
	var updateEpgWarning string
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
						Headers:   source.Headers,
						CronTime:  source.CronTime,
						Status:    true,
						IsSyncing: true,
					}
					if err := model.DB.Create(&epgSource).Error; err == nil {
						slog.Info("Auto-created EPG source from x-tvg-url (update)", "epg_id", epgSource.ID, "url", tvgURL, "live_source", source.Name)
						if epgSource.CronTime != "" {
							if epgCfg, err := task.ParseScheduleConfig(epgSource.CronTime); err == nil {
								lc.scheduler.AddEPGSourceTask(epgSource.ID, epgCfg)
							}
						}
						lc.scheduler.TriggerEPGSourceNow(epgSource.ID)
					}
				} else {
					slog.Info("Skipped auto-create EPG source (update), URL already exists", "url", tvgURL)
					updateEpgWarning = i18n.T(i18n.Lang(c), "message.epg_url_exists")
				}
			}
		}
	}

	// Update scheduler for refresh tasks
	if source.Type != model.LiveSourceTypeNetworkManual && source.CronTime != "" && source.Status {
		if cfg, err := task.ParseScheduleConfig(source.CronTime); err == nil {
			lc.scheduler.AddLiveSourceTask(source.ID, cfg)
		}
	} else {
		lc.scheduler.RemoveLiveSourceTask(source.ID)
	}

	// Update scheduler for detect tasks
	if source.CronDetect != "" && source.Status {
		if cfg, err := task.ParseScheduleConfig(source.CronDetect); err == nil {
			lc.scheduler.AddDetectTask(source.ID, cfg, source.DetectStrategy)
		}
	} else {
		lc.scheduler.RemoveDetectTask(source.ID)
	}

	publish.InvalidateAll()

	if updateEpgWarning != "" {
		c.JSON(http.StatusOK, gin.H{"data": source, "warning": updateEpgWarning})
	} else {
		c.JSON(http.StatusOK, source)
	}
}

// Delete removes a live source
// DELETE /api/live-sources/:id
func (lc *LiveSourceController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
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
				c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.live_ref_publish", pi.Name)})
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
						c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.live_epg_ref_publish", pi.Name)})
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

	// Clean up per-source IPTV mutex (no-op if source was not IPTV type)
	service.RemoveIPTVLock(sourceID)

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

	publish.InvalidateAll()
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(i18n.Lang(c), "message.live_source_deleted")})
}

// Trigger manually triggers a fetch for a live source
// POST /api/live-sources/:id/trigger
func (lc *LiveSourceController) Trigger(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	model.DB.Model(&model.LiveSource{}).Where("id = ?", uint(id)).Update("is_syncing", true)

	lc.scheduler.TriggerLiveSourceNow(uint(id))
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(i18n.Lang(c), "message.trigger_fetch")})
}

// GetChannels returns parsed channels for a live source
// GET /api/live-sources/:id/channels
func (lc *LiveSourceController) GetChannels(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	// Check if ffprobe is available before triggering detection
	if err := lc.scheduler.CheckFFprobe(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.no_ffprobe")})
		return
	}

	// Parse optional detect strategy from request body
	var req struct {
		DetectStrategy string `json:"detect_strategy"`
	}
	c.ShouldBindJSON(&req)
	strategy := req.DetectStrategy
	if strategy == "" {
		strategy = "unicast" // default
	}

	lc.scheduler.TriggerDetectNow(uint(id), strategy)
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(i18n.Lang(c), "message.trigger_detect")})
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
