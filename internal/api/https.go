package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/pkg/i18n"
)

// HTTPSController handles HTTPS configuration API
type HTTPSController struct {
	httpsSvc *service.HTTPSService
}

func NewHTTPSController(httpsSvc *service.HTTPSService) *HTTPSController {
	return &HTTPSController{httpsSvc: httpsSvc}
}

// GetHTTPSSettings returns the current HTTPS configuration
// GET /api/settings/https
func (hc *HTTPSController) GetHTTPSSettings(c *gin.Context) {
	cfg := hc.httpsSvc.LoadConfig()

	c.JSON(http.StatusOK, gin.H{
		"enabled":     cfg.Enabled,
		"port":        cfg.Port,
		"mutual_tls":  cfg.MutualTLS,
		"has_cert":    hc.httpsSvc.HasCert(),
		"has_key":     hc.httpsSvc.HasKey(),
		"has_ca_cert": hc.httpsSvc.HasCACert(),
		"http_port":   hc.httpsSvc.HTTPPort(),
	})
}

// UpdateHTTPSSettingsRequest is the request body for UpdateHTTPSSettings
type UpdateHTTPSSettingsRequest struct {
	Enabled   *bool `json:"enabled"`
	Port      *int  `json:"port"`
	MutualTLS *bool `json:"mutual_tls"`
}

// UpdateHTTPSSettings saves HTTPS configuration and restarts the server
// PUT /api/settings/https
func (hc *HTTPSController) UpdateHTTPSSettings(c *gin.Context) {
	lang := i18n.Lang(c)

	var req UpdateHTTPSSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load current config
	cfg := hc.httpsSvc.LoadConfig()

	// Apply changes
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.Port != nil {
		if *req.Port < 1 || *req.Port > 65535 {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.https_port_range")})
			return
		}
		cfg.Port = *req.Port
	}

	// Validate: HTTPS port must not conflict with HTTP port
	if cfg.Enabled {
		httpPort := hc.httpsSvc.HTTPPort()
		if httpPort > 0 && cfg.Port == httpPort {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.https_port_conflict", httpPort)})
			return
		}
	}

	if req.MutualTLS != nil {
		cfg.MutualTLS = *req.MutualTLS
	}

	// Validate: if enabling, cert and key must be present
	if cfg.Enabled {
		if !hc.httpsSvc.HasCert() {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.https_cert_missing")})
			return
		}
		if !hc.httpsSvc.HasKey() {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.https_key_missing")})
			return
		}

		// Validate cert/key pair
		if err := hc.httpsSvc.ValidateCertKeyPair(); err != nil {
			slog.Error("Invalid certificate/key pair", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.https_cert_invalid")})
			return
		}

		// Validate mutual TLS: CA cert required when mTLS is enabled
		if cfg.MutualTLS && !hc.httpsSvc.HasCACert() {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.https_ca_missing")})
			return
		}
	}

	// Save config
	hc.httpsSvc.SaveConfig(cfg)

	// Restart HTTPS server with new config
	if err := hc.httpsSvc.Restart(); err != nil {
		slog.Error("Failed to restart HTTPS server", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error.https_start_failed", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "message.https_config_saved"),
	})
}

// UploadCert handles server certificate file upload
// POST /api/settings/https/cert
func (hc *HTTPSController) UploadCert(c *gin.Context) {
	hc.handleFileUpload(c, service.ServerCertFile, "message.https_cert_uploaded")
}

// UploadKey handles server private key file upload
// POST /api/settings/https/key
func (hc *HTTPSController) UploadKey(c *gin.Context) {
	hc.handleFileUpload(c, service.ServerKeyFile, "message.https_key_uploaded")
}

// UploadCACert handles CA certificate file upload
// POST /api/settings/https/ca
func (hc *HTTPSController) UploadCACert(c *gin.Context) {
	hc.handleFileUpload(c, service.CACertFile, "message.https_ca_uploaded")
}

// DeleteCACert removes the CA certificate file
// DELETE /api/settings/https/ca
func (hc *HTTPSController) DeleteCACert(c *gin.Context) {
	lang := i18n.Lang(c)

	caPath := hc.httpsSvc.CACertPath()
	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "message.https_ca_deleted")})
		return
	}

	if err := os.Remove(caPath); err != nil {
		slog.Error("Failed to delete CA certificate", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error.save_file_failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "message.https_ca_deleted")})
}

// handleFileUpload is a shared helper for certificate/key file uploads
func (hc *HTTPSController) handleFileUpload(c *gin.Context, targetFilename string, successMsgKey string) {
	lang := i18n.Lang(c)

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.select_upload_file")})
		return
	}

	// Validate file extension (PEM format only)
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".pem": true, ".crt": true, ".cer": true, ".key": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error.https_invalid_file_type")})
		return
	}

	// Ensure certs directory exists
	certsDir := hc.httpsSvc.CertsDir()
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		slog.Error("Failed to create certs directory", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error.mkdir_failed")})
		return
	}

	targetPath := filepath.Join(certsDir, targetFilename)

	// Save uploaded file
	if err := c.SaveUploadedFile(file, targetPath); err != nil {
		slog.Error("Failed to save certificate file", "error", err, "target", targetFilename)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error.save_file_failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, successMsgKey),
	})
}
