package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
)

// LogoController handles channel logo CRUD and upload
type LogoController struct {
	logoDir string // Directory where logos are stored (e.g., "logos/")
}

func NewLogoController(logoDir string) *LogoController {
	// Ensure directory exists
	os.MkdirAll(logoDir, 0755)
	return &LogoController{logoDir: logoDir}
}

// List returns all channel logos
// GET /api/logos
func (lc *LogoController) List(c *gin.Context) {
	var logos []model.ChannelLogo
	if err := model.DB.Order("id desc").Find(&logos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logos)
}

// Upload handles logo file upload
// POST /api/logos/upload
func (lc *LogoController) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true, ".webp": true, ".ico": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type, allowed: png, jpg, jpeg, gif, svg, webp, ico"})
		return
	}

	// Generate unique filename
	name := strings.TrimSuffix(file.Filename, ext)

	// Check for logo name uniqueness
	var existing int64
	model.DB.Model(&model.ChannelLogo{}).Where("name = ?", name).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("台标名称「%s」已存在，请重命名文件后再上传", name)})
		return
	}

	fileName := fmt.Sprintf("%s_%d%s", name, model.DB.NowFunc().UnixMilli(), ext)
	filePath := filepath.Join(lc.logoDir, fileName)
	urlPath := "/logo/" + fileName

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file: " + err.Error()})
		return
	}

	logo := model.ChannelLogo{
		Name:     name,
		FilePath: filePath,
		URLPath:  urlPath,
	}

	if err := model.DB.Create(&logo).Error; err != nil {
		os.Remove(filePath) // Clean up file on DB error
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, logo)
}

// BatchUpload handles multiple logo file uploads
// POST /api/logos/batch-upload
func (lc *LogoController) BatchUpload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}

	allowedExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true, ".webp": true, ".ico": true}
	var uploaded []model.ChannelLogo
	var errors []string

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !allowedExts[ext] {
			errors = append(errors, fmt.Sprintf("%s: unsupported file type", file.Filename))
			continue
		}

		name := strings.TrimSuffix(file.Filename, ext)

		// Check for logo name uniqueness
		var existing int64
		model.DB.Model(&model.ChannelLogo{}).Where("name = ?", name).Count(&existing)
		if existing > 0 {
			errors = append(errors, fmt.Sprintf("%s: 台标名称已存在", file.Filename))
			continue
		}

		fileName := fmt.Sprintf("%s_%d%s", name, model.DB.NowFunc().UnixMilli(), ext)
		filePath := filepath.Join(lc.logoDir, fileName)
		urlPath := "/logo/" + fileName

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: save failed", file.Filename))
			continue
		}

		logo := model.ChannelLogo{
			Name:     name,
			FilePath: filePath,
			URLPath:  urlPath,
		}
		if err := model.DB.Create(&logo).Error; err != nil {
			os.Remove(filePath)
			errors = append(errors, fmt.Sprintf("%s: db error", file.Filename))
			continue
		}

		uploaded = append(uploaded, logo)
	}

	c.JSON(http.StatusOK, gin.H{
		"uploaded": uploaded,
		"errors":   errors,
	})
}

// UpdateLogoRequest is the request body for updating a logo
type UpdateLogoRequest struct {
	Name string `json:"name" binding:"required"`
}

// Update modifies a channel logo's name
// PUT /api/logos/:id
func (lc *LogoController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var req UpdateLogoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// Check name uniqueness (excluding self)
	var existing int64
	model.DB.Model(&model.ChannelLogo{}).Where("name = ? AND id != ?", req.Name, id).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该台标名称已存在，请换一个名称"})
		return
	}

	var logo model.ChannelLogo
	if err := model.DB.First(&logo, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该台标"})
		return
	}

	logo.Name = req.Name
	if err := model.DB.Save(&logo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logo)
}

// Delete removes a channel logo
// DELETE /api/logos/:id
func (lc *LogoController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var logo model.ChannelLogo
	if err := model.DB.First(&logo, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "logo not found"})
		return
	}

	// Delete file from disk
	os.Remove(logo.FilePath)

	if err := model.DB.Delete(&logo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logo deleted"})
}
