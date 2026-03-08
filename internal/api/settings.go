package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/service"
)

// SettingsController handles system settings for detection configuration
type SettingsController struct {
	dataDir       string
	detectService *service.DetectService
}

func NewSettingsController(dataDir string) *SettingsController {
	return &SettingsController{
		dataDir:       dataDir,
		detectService: service.NewDetectService(dataDir),
	}
}

// GetDetectSettings returns the current detection configuration
// GET /api/settings/detect
func (sc *SettingsController) GetDetectSettings(c *gin.Context) {
	concurrency := strconv.Itoa(service.DefaultDetectConcurrency)
	timeout := strconv.Itoa(service.DefaultDetectTimeout)

	var settings []model.SystemSetting
	model.DB.Where("key IN ?", []string{"detect_concurrency", "detect_timeout"}).Find(&settings)

	for _, s := range settings {
		switch s.Key {
		case "detect_concurrency":
			concurrency = s.Value
		case "detect_timeout":
			timeout = s.Value
		}
	}

	// Get ffprobe version if available
	ffprobeVersion := ""
	if ver, err := sc.detectService.GetFFprobeVersion(); err == nil {
		ffprobeVersion = ver
	}

	concurrencyInt, _ := strconv.Atoi(concurrency)
	timeoutInt, _ := strconv.Atoi(timeout)

	c.JSON(http.StatusOK, gin.H{
		"concurrency":     concurrencyInt,
		"timeout":         timeoutInt,
		"ffprobe_version": ffprobeVersion,
	})
}

// UpdateDetectSettings updates the detection configuration
// PUT /api/settings/detect
func (sc *SettingsController) UpdateDetectSettings(c *gin.Context) {
	var req struct {
		Concurrency *int `json:"concurrency"`
		Timeout     *int `json:"timeout"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Concurrency != nil {
		if *req.Concurrency < 1 || *req.Concurrency > 30 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "检测并发数范围为 1-30"})
			return
		}
		sc.upsertSetting("detect_concurrency", strconv.Itoa(*req.Concurrency))
	}

	if req.Timeout != nil {
		if *req.Timeout < 1 || *req.Timeout > 30 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "检测超时范围为 1-30 秒"})
			return
		}
		sc.upsertSetting("detect_timeout", strconv.Itoa(*req.Timeout))
	}

	c.JSON(http.StatusOK, gin.H{"message": "检测配置已更新"})
}

// UploadFFprobe handles ffprobe executable file upload
// POST /api/settings/detect/ffprobe
func (sc *SettingsController) UploadFFprobe(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
		return
	}

	// Determine target filename
	targetName := "ffprobe"
	if runtime.GOOS == "windows" {
		targetName = "ffprobe.exe"
	}

	// Ensure detect directory exists
	detectDir := filepath.Join(sc.dataDir, "detect")
	if err := os.MkdirAll(detectDir, 0755); err != nil {
		slog.Error("Failed to create detect directory", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
		return
	}

	targetPath := filepath.Join(detectDir, targetName)

	// Save uploaded file
	if err := c.SaveUploadedFile(file, targetPath); err != nil {
		slog.Error("Failed to save ffprobe file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	// Set executable permission (Unix-like systems)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0755); err != nil {
			slog.Warn("Failed to set executable permission", "error", err)
		}
	}

	// Verify the uploaded file is actually ffprobe
	version, err := sc.detectService.GetFFprobeVersion()
	if err != nil {
		// Remove invalid file
		os.Remove(targetPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("上传的文件不是有效的 ffprobe 可执行文件: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "ffprobe 上传成功",
		"ffprobe_version": version,
	})
}

// upsertSetting creates or updates a system setting
func (sc *SettingsController) upsertSetting(key, value string) {
	var setting model.SystemSetting
	result := model.DB.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		// Create new
		model.DB.Create(&model.SystemSetting{Key: key, Value: value})
	} else {
		// Update existing
		model.DB.Model(&setting).Update("value", value)
	}
}
