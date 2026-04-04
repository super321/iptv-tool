package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/publish"
	"iptv-tool-v2/internal/task"
	"iptv-tool-v2/pkg/i18n"
)

// PublishController handles CRUD for publish interfaces
type PublishController struct {
	scheduler *task.Scheduler
}

func NewPublishController(scheduler *task.Scheduler) *PublishController {
	return &PublishController{scheduler: scheduler}
}

// ListInterfaces returns all publish interfaces
// GET /api/publish
func (pc *PublishController) ListInterfaces(c *gin.Context) {
	var interfaces []model.PublishInterface
	if err := model.DB.Order("id desc").Find(&interfaces).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, interfaces)
}

// GetInterface returns a single publish interface
// GET /api/publish/:id
func (pc *PublishController) GetInterface(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	var iface model.PublishInterface
	if err := model.DB.First(&iface, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.publish_not_found")})
		return
	}
	c.JSON(http.StatusOK, iface)
}

// CreateInterfaceRequest is the request body for creating a publish interface
type CreateInterfaceRequest struct {
	Name                   string              `json:"name" binding:"required"`
	Description            string              `json:"description"`
	Path                   string              `json:"path" binding:"required"`
	Type                   string              `json:"type" binding:"required,oneof=live epg"`
	Format                 model.PublishFormat `json:"format" binding:"required"`
	SourceIDs              string              `json:"source_ids"`
	RuleIDs                string              `json:"rule_ids"`
	TvgIDMode              string              `json:"tvg_id_mode"`
	EPGDays                int                 `json:"epg_days"`
	GzipEnabled            bool                `json:"gzip_enabled"`
	AddressType            string              `json:"address_type"`
	MulticastType          string              `json:"multicast_type"`
	UDPxyURL               string              `json:"udpxy_url"`
	FCCEnabled             bool                `json:"fcc_enabled"`
	FCCType                string              `json:"fcc_type"`
	CustomParams           string              `json:"custom_params"`
	M3UCatchupTemplate     string              `json:"m3u_catchup_template"`
	UnicastType            string              `json:"unicast_type"`
	UnicastProxyRules      string              `json:"unicast_proxy_rules"`
	FilterInvalidSourceIDs string              `json:"filter_invalid_source_ids"`
	SourceOutputConfigs    string              `json:"source_output_configs"`
	UACheckEnabled         bool                `json:"ua_check_enabled"`
	UAAllowedValues        string              `json:"ua_allowed_values"`
}

// CreateInterface adds a new publish interface
// POST /api/publish
func (pc *PublishController) CreateInterface(c *gin.Context) {
	var req CreateInterfaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Trim whitespace from string inputs
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Path = strings.TrimSpace(req.Path)
	req.UDPxyURL = strings.TrimSpace(req.UDPxyURL)
	req.M3UCatchupTemplate = strings.TrimSpace(req.M3UCatchupTemplate)

	// Validate format based on type
	if req.Type == "live" && (req.Format != model.PublishFormatM3U && req.Format != model.PublishFormatTXT) {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.live_format_invalid")})
		return
	}
	if req.Type == "epg" && (req.Format != model.PublishFormatXMLTV && req.Format != model.PublishFormatDIYP) {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.epg_format_invalid")})
		return
	}

	// Check name uniqueness
	var existingName int64
	model.DB.Model(&model.PublishInterface{}).Where("name = ?", req.Name).Count(&existingName)
	if existingName > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.interface_name_exists")})
		return
	}

	// Check path uniqueness
	var existing int64
	model.DB.Model(&model.PublishInterface{}).Where("path = ? AND type = ?", req.Path, req.Type).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.path_exists")})
		return
	}

	iface := model.PublishInterface{
		Name:                   req.Name,
		Description:            req.Description,
		Path:                   req.Path,
		Type:                   req.Type,
		Format:                 req.Format,
		SourceIDs:              req.SourceIDs,
		RuleIDs:                req.RuleIDs,
		TvgIDMode:              req.TvgIDMode,
		Status:                 true,
		EPGDays:                req.EPGDays,
		GzipEnabled:            req.GzipEnabled,
		AddressType:            req.AddressType,
		MulticastType:          req.MulticastType,
		UDPxyURL:               req.UDPxyURL,
		FCCEnabled:             req.FCCEnabled,
		FCCType:                req.FCCType,
		CustomParams:           req.CustomParams,
		M3UCatchupTemplate:     req.M3UCatchupTemplate,
		UnicastType:            req.UnicastType,
		UnicastProxyRules:      req.UnicastProxyRules,
		FilterInvalidSourceIDs: req.FilterInvalidSourceIDs,
		SourceOutputConfigs:    req.SourceOutputConfigs,
		UACheckEnabled:         req.UACheckEnabled,
		UAAllowedValues:        req.UAAllowedValues,
	}

	if err := model.DB.Create(&iface).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	publish.InvalidateAll()
	c.JSON(http.StatusCreated, iface)
}

// UpdateInterfaceRequest is the request body for updating a publish interface
type UpdateInterfaceRequest struct {
	Name                   *string              `json:"name"`
	Description            *string              `json:"description"`
	Path                   *string              `json:"path"`
	Format                 *model.PublishFormat `json:"format"`
	SourceIDs              *string              `json:"source_ids"`
	RuleIDs                *string              `json:"rule_ids"`
	TvgIDMode              *string              `json:"tvg_id_mode"`
	Status                 *bool                `json:"status"`
	EPGDays                *int                 `json:"epg_days"`
	GzipEnabled            *bool                `json:"gzip_enabled"`
	AddressType            *string              `json:"address_type"`
	MulticastType          *string              `json:"multicast_type"`
	UDPxyURL               *string              `json:"udpxy_url"`
	FCCEnabled             *bool                `json:"fcc_enabled"`
	FCCType                *string              `json:"fcc_type"`
	CustomParams           *string              `json:"custom_params"`
	M3UCatchupTemplate     *string              `json:"m3u_catchup_template"`
	UnicastType            *string              `json:"unicast_type"`
	UnicastProxyRules      *string              `json:"unicast_proxy_rules"`
	FilterInvalidSourceIDs *string              `json:"filter_invalid_source_ids"`
	SourceOutputConfigs    *string              `json:"source_output_configs"`
	UACheckEnabled         *bool                `json:"ua_check_enabled"`
	UAAllowedValues        *string              `json:"ua_allowed_values"`
}

// UpdateInterface modifies a publish interface
// PUT /api/publish/:id
func (pc *PublishController) UpdateInterface(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	var iface model.PublishInterface
	if err := model.DB.First(&iface, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.publish_not_found")})
		return
	}

	var req UpdateInterfaceRequest
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
	if req.Path != nil {
		*req.Path = strings.TrimSpace(*req.Path)
	}
	if req.UDPxyURL != nil {
		*req.UDPxyURL = strings.TrimSpace(*req.UDPxyURL)
	}
	if req.M3UCatchupTemplate != nil {
		*req.M3UCatchupTemplate = strings.TrimSpace(*req.M3UCatchupTemplate)
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		var existingName int64
		model.DB.Model(&model.PublishInterface{}).Where("name = ? AND id != ?", *req.Name, id).Count(&existingName)
		if existingName > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.interface_name_exists")})
			return
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Path != nil {
		// Check uniqueness
		var existing int64
		// Combine type and path for uniqueness check
		interfaceType := iface.Type // Default to existing type
		model.DB.Model(&model.PublishInterface{}).Where("path = ? AND type = ? AND id != ?", *req.Path, interfaceType, id).Count(&existing)
		if existing > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.path_exists")})
			return
		}
		updates["path"] = *req.Path
	}
	if req.Format != nil {
		updates["format"] = *req.Format
	}
	if req.SourceIDs != nil {
		updates["source_ids"] = *req.SourceIDs
	}
	if req.RuleIDs != nil {
		updates["rule_ids"] = *req.RuleIDs
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.EPGDays != nil {
		updates["epg_days"] = *req.EPGDays
	}
	if req.GzipEnabled != nil {
		updates["gzip_enabled"] = *req.GzipEnabled
	}
	if req.TvgIDMode != nil {
		updates["tvg_id_mode"] = *req.TvgIDMode
	}
	if req.AddressType != nil {
		updates["address_type"] = *req.AddressType
	}
	if req.MulticastType != nil {
		updates["multicast_type"] = *req.MulticastType
	}
	if req.UDPxyURL != nil {
		updates["udpxy_url"] = *req.UDPxyURL
	}
	if req.FCCEnabled != nil {
		updates["fcc_enabled"] = *req.FCCEnabled
	}
	if req.FCCType != nil {
		updates["fcc_type"] = *req.FCCType
	}
	if req.CustomParams != nil {
		updates["custom_params"] = *req.CustomParams
	}
	if req.M3UCatchupTemplate != nil {
		updates["m3u_catchup_template"] = *req.M3UCatchupTemplate
	}
	if req.FilterInvalidSourceIDs != nil {
		updates["filter_invalid_source_ids"] = *req.FilterInvalidSourceIDs
	}
	if req.SourceOutputConfigs != nil {
		updates["source_output_configs"] = *req.SourceOutputConfigs
	}
	if req.UnicastType != nil {
		updates["unicast_type"] = *req.UnicastType
	}
	if req.UnicastProxyRules != nil {
		updates["unicast_proxy_rules"] = *req.UnicastProxyRules
	}
	if req.UACheckEnabled != nil {
		updates["ua_check_enabled"] = *req.UACheckEnabled
	}
	if req.UAAllowedValues != nil {
		updates["ua_allowed_values"] = *req.UAAllowedValues
	}

	if len(updates) > 0 {
		if err := model.DB.Model(&iface).Updates(updates).Error; err != nil {
			slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	model.DB.First(&iface, uint(id))
	publish.InvalidateAll()
	c.JSON(http.StatusOK, iface)
}

// DeleteInterface removes a publish interface
// DELETE /api/publish/:id
func (pc *PublishController) DeleteInterface(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	if err := model.DB.Delete(&model.PublishInterface{}, uint(id)).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	publish.InvalidateAll()
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(i18n.Lang(c), "message.publish_deleted")})
}

// PreviewRequest is the request body for previewing the publish output
type PreviewRequest struct {
	Type                   string `json:"type" binding:"required"`
	SourceIDs              string `json:"source_ids"`
	RuleIDs                string `json:"rule_ids"`
	AddressType            string `json:"address_type"`
	MulticastType          string `json:"multicast_type"`
	UDPxyURL               string `json:"udpxy_url"`
	FCCEnabled             bool   `json:"fcc_enabled"`
	FCCType                string `json:"fcc_type"`
	CustomParams           string `json:"custom_params"`
	UnicastType            string `json:"unicast_type"`
	UnicastProxyRules      string `json:"unicast_proxy_rules"`
	FilterInvalidSourceIDs string `json:"filter_invalid_source_ids"`
	SourceOutputConfigs    string `json:"source_output_configs"`
}

// PreviewInterface generates a dry-run preview of the aggregated channels/epgs
// POST /api/publish/preview
func (pc *PublishController) PreviewInterface(c *gin.Context) {
	var req PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a dummy interface purely for the engine to consume IDs
	dummyIface := model.PublishInterface{
		Type:                   req.Type,
		SourceIDs:              req.SourceIDs,
		RuleIDs:                req.RuleIDs,
		AddressType:            req.AddressType,
		MulticastType:          req.MulticastType,
		UDPxyURL:               req.UDPxyURL,
		FCCEnabled:             req.FCCEnabled,
		FCCType:                req.FCCType,
		CustomParams:           req.CustomParams,
		UnicastType:            req.UnicastType,
		UnicastProxyRules:      req.UnicastProxyRules,
		FilterInvalidSourceIDs: req.FilterInvalidSourceIDs,
		SourceOutputConfigs:    req.SourceOutputConfigs,
	}

	eng, err := publish.NewEngine(dummyIface)
	if err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Type == "live" {
		channels, err := eng.AggregateLiveChannels()
		if err != nil {
			slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, channels)
	} else {
		epg, err := eng.AggregateEPG()
		if err != nil {
			slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Map EPGs back to channels and count them for simple preview
		type EPGPreviewRow struct {
			ChannelID    string `json:"channel_id"`
			OriginalName string `json:"original_name"`
			Alias        string `json:"alias"`
			ProgramCount int    `json:"program_count"`
		}

		var result []EPGPreviewRow
		if epg != nil {
			for _, key := range epg.ChannelOrder {
				chEntry := epg.Channels[key]
				count := 0
				for _, progs := range chEntry.DatePrograms {
					count += len(progs)
				}
				result = append(result, EPGPreviewRow{
					ChannelID:    chEntry.ChannelID,
					OriginalName: chEntry.ChannelName,
					Alias:        chEntry.Alias,
					ProgramCount: count,
				})
			}
		}

		c.JSON(http.StatusOK, result)
	}
}

// DownloadInterface serves the publish content for admin download (no UA check)
// GET /api/publish/:id/download
func (pc *PublishController) DownloadInterface(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	var iface model.PublishInterface
	if err := model.DB.First(&iface, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.publish_not_found")})
		return
	}

	engine, err := publish.NewEngine(iface)
	if err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	requestHost := c.Request.Host
	if fwd := c.GetHeader("X-Forwarded-Host"); fwd != "" {
		requestHost = fwd
	}

	// Serve content directly, bypassing UA check (admin is authenticated via JWT)
	publish.ServeLiveOrEPG(c, engine, iface, requestHost)
}
