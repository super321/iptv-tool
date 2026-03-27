package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/publish"
	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/internal/task"
	"iptv-tool-v2/internal/version"
	"iptv-tool-v2/pkg/i18n"
)

// ConfigTransferController handles config export/import API endpoints.
type ConfigTransferController struct {
	svc *service.ConfigTransferService
}

// NewConfigTransferController creates a new controller.
func NewConfigTransferController(scheduler *task.Scheduler, logoDir string) *ConfigTransferController {
	return &ConfigTransferController{
		svc: service.NewConfigTransferService(logoDir, scheduler),
	}
}

// ExportRequest is the request body for the export endpoint.
type ExportRequest struct {
	Modules []string `json:"modules" binding:"required"`
}

// Export handles POST /api/config/export — generates and streams a ZIP.
func (cc *ConfigTransferController) Export(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Modules) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.config_no_modules")})
		return
	}

	buf, err := cc.svc.ExportConfig(req.Modules)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("%s: %s", i18n.T(i18n.Lang(c), "error.config_export_failed"), err.Error())})
		return
	}

	filename := fmt.Sprintf("iptv-config-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

// ImportParse handles POST /api/config/import/parse — parses ZIP and returns summary.
func (cc *ConfigTransferController) ImportParse(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.select_upload_file")})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.config_open_file")})
		return
	}
	defer f.Close()

	zipData, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.config_read_file")})
		return
	}

	parsed, err := cc.svc.ParseImportZip(zipData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", i18n.T(i18n.Lang(c), "error.config_parse_failed"), err.Error())})
		return
	}

	response := gin.H{
		"manifest":  parsed.Manifest,
		"summaries": parsed.Summaries,
	}

	// Add version mismatch warning if applicable
	if parsed.Manifest.Version != version.Version {
		response["version_warning"] = i18n.T(i18n.Lang(c), "message.config_version_mismatch", parsed.Manifest.Version, version.Version)
	}

	c.JSON(http.StatusOK, response)
}

// ImportExecute handles POST /api/config/import/execute — parses ZIP again and executes import.
func (cc *ConfigTransferController) ImportExecute(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.select_upload_file")})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.config_open_file")})
		return
	}
	defer f.Close()

	zipData, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(i18n.Lang(c), "error.config_read_file")})
		return
	}

	parsed, err := cc.svc.ParseImportZip(zipData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s: %s", i18n.T(i18n.Lang(c), "error.config_parse_failed"), err.Error())})
		return
	}

	result := cc.svc.ExecuteImport(parsed)
	publish.InvalidateAll()
	c.JSON(http.StatusOK, result)
}
