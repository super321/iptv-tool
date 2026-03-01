package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
)

// RuleController handles CRUD for aggregation rules
type RuleController struct{}

func NewRuleController() *RuleController {
	return &RuleController{}
}

func (rc *RuleController) List(c *gin.Context) {
	var rules []model.AggregationRule
	if err := model.DB.Order("id desc").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (rc *RuleController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var rule model.AggregationRule
	if err := model.DB.First(&rule, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该规则"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效: " + err.Error()})
		return
	}

	// 【新增】检查规则名称是否已存在
	var existing int64
	model.DB.Model(&model.AggregationRule{}).Where("name = ?", req.Name).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该规则名称已存在，请换一个名称"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var rule model.AggregationRule
	if err := model.DB.First(&rule, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该规则"})
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效: " + err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		// 【新增】检查更新的新名称是否和其他规则重名
		var existing int64
		model.DB.Model(&model.AggregationRule{}).Where("name = ? AND id != ?", *req.Name, id).Count(&existing)
		if existing > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "该规则名称已存在，请换一个名称"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	model.DB.First(&rule, uint(id))
	c.JSON(http.StatusOK, rule)
}

func (rc *RuleController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	ruleIDStr := strconv.FormatUint(id, 10)

	// 检查该规则是否被发布接口引用
	var interfaces []model.PublishInterface
	if err := model.DB.Find(&interfaces).Error; err == nil {
		for _, pi := range interfaces {
			if pi.RuleIDs == "" {
				continue
			}
			for _, rID := range strings.Split(pi.RuleIDs, ",") {
				if strings.TrimSpace(rID) == ruleIDStr {
					c.JSON(http.StatusConflict, gin.H{"error": "该规则正被发布接口「" + pi.Name + "」引用，请先移除引用后再删除"})
					return
				}
			}
		}
	}

	if err := model.DB.Delete(&model.AggregationRule{}, uint(id)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "规则已删除"})
}
