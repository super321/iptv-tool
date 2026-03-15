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
