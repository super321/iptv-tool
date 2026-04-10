package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/publish"
	"iptv-tool-v2/pkg/i18n"
)

// RuleController handles CRUD for aggregation rules
type RuleController struct{}

func NewRuleController() *RuleController {
	return &RuleController{}
}

func (rc *RuleController) List(c *gin.Context) {
	var rules []model.AggregationRule
	if err := model.DB.Order("id desc").Find(&rules).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (rc *RuleController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	var rule model.AggregationRule
	if err := model.DB.First(&rule, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.rule_not_found")})
		return
	}
	c.JSON(http.StatusOK, rule)
}

type CreateRuleRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Type        model.RuleType  `json:"type" binding:"required,oneof=alias filter group"`
	Config      json.RawMessage `json:"config" binding:"required"`
}

func (rc *RuleController) Create(c *gin.Context) {
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_params") + ": " + err.Error()})
		return
	}

	// Trim whitespace from string inputs
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)

	// Check rule name uniqueness
	var existing int64
	model.DB.Model(&model.AggregationRule{}).Where("name = ?", req.Name).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.rule_name_exists")})
		return
	}

	rule := model.AggregationRule{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      string(req.Config),
		Status:      true,
	}

	if err := model.DB.Create(&rule).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	publish.InvalidateAll()
	c.JSON(http.StatusCreated, rule)
}

type UpdateRuleRequest struct {
	Name        *string          `json:"name"`
	Description *string          `json:"description"`
	Config      *json.RawMessage `json:"config"`
	Status      *bool            `json:"status"`
}

func (rc *RuleController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	var rule model.AggregationRule
	if err := model.DB.First(&rule, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.rule_not_found")})
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_params") + ": " + err.Error()})
		return
	}

	// Trim whitespace from string inputs
	if req.Name != nil {
		*req.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		*req.Description = strings.TrimSpace(*req.Description)
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		// Check rule name uniqueness (excluding self)
		var existing int64
		model.DB.Model(&model.AggregationRule{}).Where("name = ? AND id != ?", *req.Name, id).Count(&existing)
		if existing > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.rule_name_exists")})
			return
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Config != nil {
		updates["config"] = string(*req.Config)
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		if err := model.DB.Model(&rule).Updates(updates).Error; err != nil {
			slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	model.DB.First(&rule, uint(id))
	publish.InvalidateAll()
	c.JSON(http.StatusOK, rule)
}

func (rc *RuleController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_id")})
		return
	}

	ruleIDStr := strconv.FormatUint(id, 10)

	// Check if rule is referenced by any publish interface
	var interfaces []model.PublishInterface
	if err := model.DB.Find(&interfaces).Error; err == nil {
		for _, pi := range interfaces {
			if pi.RuleIDs == "" {
				continue
			}
			for _, rID := range strings.Split(pi.RuleIDs, ",") {
				if strings.TrimSpace(rID) == ruleIDStr {
					c.JSON(http.StatusConflict, gin.H{"error": i18n.T(i18n.Lang(c), "error.rule_ref_publish", pi.Name)})
					return
				}
			}
		}
	}

	if err := model.DB.Delete(&model.AggregationRule{}, uint(id)).Error; err != nil {
		slog.Error("Internal server error", "error", err, "path", c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	publish.InvalidateAll()
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(i18n.Lang(c), "message.rule_deleted")})
}

// TestRuleRequest is the request body for testing a rule before saving
type TestRuleRequest struct {
	Type       model.RuleType  `json:"type" binding:"required,oneof=alias filter group"`
	Config     json.RawMessage `json:"config" binding:"required"`
	SourceType string          `json:"source_type" binding:"required,oneof=live epg"`
	SourceIDs  []uint          `json:"source_ids" binding:"required,min=1"`
}

// TestRuleOriginalItem represents a single item in the original data
type TestRuleOriginalItem struct {
	Name  string `json:"name"`
	Group string `json:"group"`
}

// TestRuleAppliedItem represents a single item after rule application
type TestRuleAppliedItem struct {
	Name   string `json:"name"`
	Alias  string `json:"alias"`
	Group  string `json:"group"`
	Status string `json:"status"` // "modified", "filtered", "unchanged"
}

// TestRuleSummary holds aggregated counts
type TestRuleSummary struct {
	Total     int `json:"total"`
	Modified  int `json:"modified"`
	Filtered  int `json:"filtered"`
	Unchanged int `json:"unchanged"`
}

// TestRule applies unsaved rule config to selected data sources and returns a diff
// POST /api/rules/test
func (rc *RuleController) TestRule(c *gin.Context) {
	lang := i18n.Lang(c)

	var req TestRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.invalid_params") + ": " + err.Error()})
		return
	}

	// Compile rules from the unsaved config
	engine, err := publish.NewTestEngine(req.Type, req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.test_rule_invalid_config") + ": " + err.Error()})
		return
	}

	// Load channel data from sources
	type channelItem struct {
		Name  string
		Group string
	}
	var items []channelItem

	if req.SourceType == "live" {
		var channels []model.ParsedChannel
		if err := model.DB.Where("source_id IN ?", req.SourceIDs).Find(&channels).Error; err != nil {
			slog.Error("TestRule: failed to load live channels", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Deduplicate by name to reduce noise
		seen := make(map[string]bool)
		for _, ch := range channels {
			if !seen[ch.Name] {
				seen[ch.Name] = true
				items = append(items, channelItem{Name: ch.Name, Group: ch.Group})
			}
		}
	} else {
		// EPG source: load distinct channel names
		type epgChannel struct {
			ChannelName string `gorm:"column:channel_name"`
		}
		var epgChannels []epgChannel
		if err := model.DB.Model(&model.ParsedEPG{}).
			Select("DISTINCT channel_name").
			Where("source_id IN ?", req.SourceIDs).
			Find(&epgChannels).Error; err != nil {
			slog.Error("TestRule: failed to load EPG channels", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, ch := range epgChannels {
			if ch.ChannelName != "" {
				items = append(items, channelItem{Name: ch.ChannelName, Group: ""})
			}
		}
	}

	if len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.test_rule_no_sources")})
		return
	}

	// Apply rules and build diff
	original := make([]TestRuleOriginalItem, 0, len(items))
	applied := make([]TestRuleAppliedItem, 0, len(items))
	summary := TestRuleSummary{Total: len(items)}

	for _, item := range items {
		orig := TestRuleOriginalItem{
			Name:  item.Name,
			Group: item.Group,
		}
		original = append(original, orig)

		result := engine.TestApplyRules(item.Name, item.Group, req.SourceType == "epg")

		appliedItem := TestRuleAppliedItem{
			Name:  item.Name,
			Alias: result.Alias,
			Group: result.Group,
		}

		if result.Filtered {
			appliedItem.Status = "filtered"
			summary.Filtered++
		} else if result.Alias != "" || result.Group != item.Group {
			appliedItem.Status = "modified"
			summary.Modified++
		} else {
			appliedItem.Status = "unchanged"
			summary.Unchanged++
		}

		applied = append(applied, appliedItem)
	}

	c.JSON(http.StatusOK, gin.H{
		"original": original,
		"applied":  applied,
		"summary":  summary,
	})
}
