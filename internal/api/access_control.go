package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/pkg/i18n"
)

// AccessControlController handles access control settings
type AccessControlController struct{}

func NewAccessControlController() *AccessControlController {
	return &AccessControlController{}
}

// AccessControlResponse is the response for GET access control settings
type AccessControlResponse struct {
	Mode    string                     `json:"mode"`
	Entries []model.AccessControlEntry `json:"entries"`
}

// UpdateAccessControlRequest is the request for updating access control settings
type UpdateAccessControlRequest struct {
	Mode    string                      `json:"mode" binding:"required,oneof=disabled whitelist blacklist"`
	Entries []AccessControlEntryRequest `json:"entries"`
}

// AccessControlEntryRequest represents a single entry in the update request
type AccessControlEntryRequest struct {
	EntryType string `json:"entry_type" binding:"required,oneof=single cidr range"`
	Value     string `json:"value" binding:"required"`
	BlockDays *int   `json:"block_days"` // nil = permanent (blacklist only)
}

// validateEntryValue validates the IP/CIDR/range format of an entry using net package.
func validateEntryValue(entry AccessControlEntryRequest) error {
	value := strings.TrimSpace(entry.Value)
	if value == "" {
		return fmt.Errorf("value is empty")
	}

	switch entry.EntryType {
	case "single":
		if net.ParseIP(value) == nil {
			return fmt.Errorf("invalid IP address: %s", value)
		}
	case "cidr":
		_, _, err := net.ParseCIDR(value)
		if err != nil {
			return fmt.Errorf("invalid CIDR: %s", value)
		}
	case "range":
		parts := strings.SplitN(value, "~", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid IP range format: %s (expected start~end)", value)
		}
		startIP := net.ParseIP(strings.TrimSpace(parts[0]))
		endIP := net.ParseIP(strings.TrimSpace(parts[1]))
		if startIP == nil {
			return fmt.Errorf("invalid start IP in range: %s", parts[0])
		}
		if endIP == nil {
			return fmt.Errorf("invalid end IP in range: %s", parts[1])
		}
		// Ensure start <= end
		startB := startIP.To16()
		endB := endIP.To16()
		if bytesCompare(startB, endB) > 0 {
			return fmt.Errorf("start IP must not be greater than end IP: %s ~ %s", parts[0], parts[1])
		}
	}
	return nil
}

// GetAccessControl returns the current access control settings
// GET /api/settings/access-control
func (ac *AccessControlController) GetAccessControl(c *gin.Context) {
	// Read mode from system settings
	mode := "disabled"
	var setting model.SystemSetting
	if err := model.DB.Where("key = ?", "access_control_mode").First(&setting).Error; err == nil {
		mode = setting.Value
	}

	// Read entries
	var entries []model.AccessControlEntry
	if mode == "whitelist" {
		model.DB.Where("list_type = ?", "whitelist").Find(&entries)
	} else if mode == "blacklist" {
		model.DB.Where("list_type = ?", "blacklist").Find(&entries)
	}

	c.JSON(http.StatusOK, AccessControlResponse{
		Mode:    mode,
		Entries: entries,
	})
}

// UpdateAccessControl saves the access control mode and entries
// PUT /api/settings/access-control
func (ac *AccessControlController) UpdateAccessControl(c *gin.Context) {
	var req UpdateAccessControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lang := i18n.Lang(c)
	clientIP := c.ClientIP()

	// Validate all entry values before processing
	if req.Mode == "whitelist" || req.Mode == "blacklist" {
		for i, e := range req.Entries {
			if err := validateEntryValue(e); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("%s (#%d: %s)", i18n.T(lang, "error.acl_invalid_entry"), i+1, err.Error()),
				})
				return
			}
			// Blacklist only supports single IP
			if req.Mode == "blacklist" && e.EntryType != "single" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("%s (#%d)", i18n.T(lang, "error.acl_blacklist_single_only"), i+1),
				})
				return
			}
		}
	}

	// Self-lockout validation: build temporary entry list and check
	if req.Mode == "whitelist" || req.Mode == "blacklist" {
		listType := req.Mode
		tempEntries := make([]model.AccessControlEntry, 0, len(req.Entries))
		for _, e := range req.Entries {
			tempEntries = append(tempEntries, model.AccessControlEntry{
				ListType:  listType,
				EntryType: e.EntryType,
				Value:     e.Value,
				BlockDays: e.BlockDays,
			})
		}

		if !IsIPAllowed(clientIP, req.Mode, tempEntries) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": i18n.T(lang, "error.acl_self_lockout"),
			})
			return
		}
	}

	// Save mode to SystemSetting
	var setting model.SystemSetting
	result := model.DB.Where("key = ?", "access_control_mode").First(&setting)
	if result.Error != nil {
		model.DB.Create(&model.SystemSetting{Key: "access_control_mode", Value: req.Mode})
	} else {
		model.DB.Model(&setting).Update("value", req.Mode)
	}

	// Determine the list type for entries
	listType := req.Mode // "whitelist" or "blacklist"

	if req.Mode == "disabled" {
		// When disabled, clear all entries
		model.DB.Where("1 = 1").Delete(&model.AccessControlEntry{})
	} else {
		// Replace entries: delete old ones of this list type, insert new ones
		model.DB.Where("list_type = ?", listType).Delete(&model.AccessControlEntry{})
		// Also clear entries of the other list type (mode switch)
		otherType := "blacklist"
		if listType == "blacklist" {
			otherType = "whitelist"
		}
		model.DB.Where("list_type = ?", otherType).Delete(&model.AccessControlEntry{})

		// Insert new entries
		for _, e := range req.Entries {
			entry := model.AccessControlEntry{
				ListType:  listType,
				EntryType: e.EntryType,
				Value:     e.Value,
				BlockDays: e.BlockDays,
			}
			model.DB.Create(&entry)
		}
	}

	// Invalidate cache so middleware picks up changes immediately
	globalACLCache.invalidate()

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "message.acl_updated"),
	})
}

// DeleteEntry deletes a single access control entry by ID
// DELETE /api/settings/access-control/entries/:id
func (ac *AccessControlController) DeleteEntry(c *gin.Context) {
	id := c.Param("id")
	var entry model.AccessControlEntry
	if err := model.DB.First(&entry, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.Lang(c), "error.not_found")})
		return
	}

	// Check self-lockout: simulate removal
	lang := i18n.Lang(c)
	clientIP := c.ClientIP()
	mode := "disabled"
	var setting model.SystemSetting
	if err := model.DB.Where("key = ?", "access_control_mode").First(&setting).Error; err == nil {
		mode = setting.Value
	}

	if mode == "whitelist" || mode == "blacklist" {
		var remaining []model.AccessControlEntry
		model.DB.Where("list_type = ? AND id != ?", entry.ListType, entry.ID).Find(&remaining)
		if !IsIPAllowed(clientIP, mode, remaining) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": i18n.T(lang, "error.acl_self_lockout"),
			})
			return
		}
	}

	model.DB.Delete(&entry)
	globalACLCache.invalidate()

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "message.acl_entry_deleted"),
	})
}
