package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/version"
	"iptv-tool-v2/pkg/i18n"
)

const (
	githubReleaseURL = "https://api.github.com/repos/super321/iptv-tool/releases/latest"
	releasePageURL   = "https://github.com/super321/iptv-tool/releases/latest"
)

// githubRelease represents the relevant fields from the GitHub API response
type githubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

// CheckUpdate queries the GitHub API for the latest release and compares it
// with the current version.
// GET /api/system/check-update
func CheckUpdate(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubReleaseURL, nil)
	if err != nil {
		slog.Error("failed to create request for update check", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(i18n.Lang(c), "error.check_update_failed")})
		return
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("iptv-tool/%s", version.Version))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to fetch latest release from GitHub", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(i18n.Lang(c), "error.check_update_failed")})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		slog.Error("GitHub API returned non-200 status", "status", resp.StatusCode, "body", string(body))
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(i18n.Lang(c), "error.check_update_failed")})
		return
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		slog.Error("failed to decode GitHub release response", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(i18n.Lang(c), "error.check_update_failed")})
		return
	}

	hasUpdate := version.CompareVersions(version.Version, release.TagName) < 0

	c.JSON(http.StatusOK, gin.H{
		"has_update":      hasUpdate,
		"current_version": version.Version,
		"latest_version":  release.TagName,
		"release_notes":   release.Body,
		"release_url":     releasePageURL,
	})
}
