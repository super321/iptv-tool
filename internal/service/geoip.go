package service

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang/v2"

	"iptv-tool-v2/internal/model"
)

const (
	geoipDownloadURL = "https://github.com/super321/GeoLite/releases/latest/download/GeoLite2-City.tar.gz"
	geoipDBFilename  = "GeoLite2-City.mmdb"
	geoipDirName     = "geoip"

	// Download parameters
	downloadIdleTimeout = 30 * time.Second // timeout if no new bytes in 30s
	downloadMaxRetries  = 3
	progressLogStep     = 5 // log every 5% progress
)

// DownloadProgress holds the current download state for the API
type DownloadProgress struct {
	Downloading     bool   `json:"downloading"`
	DownloadedBytes int64  `json:"downloaded_bytes"`
	TotalBytes      int64  `json:"total_bytes"`
	Percent         string `json:"percent"`
	Attempt         int    `json:"attempt"`
	MaxRetries      int    `json:"max_retries"`
	Error           string `json:"error,omitempty"`
}

// GeoIPService manages the GeoIP database lifecycle
type GeoIPService struct {
	dataDir string
	dbPath  string

	mu     sync.RWMutex
	reader *geoip2.Reader

	// Download progress state
	dlMu         sync.RWMutex
	downloading  bool
	dlDownloaded int64
	dlTotal      int64
	dlAttempt    int
	dlError      string
}

// NewGeoIPService creates a new GeoIP service
func NewGeoIPService(dataDir string) *GeoIPService {
	geoipDir := filepath.Join(dataDir, geoipDirName)
	os.MkdirAll(geoipDir, 0755)

	svc := &GeoIPService{
		dataDir: dataDir,
		dbPath:  filepath.Join(geoipDir, geoipDBFilename),
	}

	// Try to open existing database
	svc.openReader()
	return svc
}

// openReader attempts to open the mmdb reader. Caller should NOT hold any lock.
func (s *GeoIPService) openReader() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.reader != nil {
		s.reader.Close()
		s.reader = nil
	}

	if _, err := os.Stat(s.dbPath); err != nil {
		return
	}

	reader, err := geoip2.Open(s.dbPath)
	if err != nil {
		slog.Error("Failed to open GeoIP database", "path", s.dbPath, "error", err)
		return
	}
	s.reader = reader
	slog.Info("GeoIP database loaded", "path", s.dbPath)
}

// Close releases the GeoIP reader
func (s *GeoIPService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reader != nil {
		s.reader.Close()
		s.reader = nil
	}
}

// DBExists returns whether the GeoIP database file exists
func (s *GeoIPService) DBExists() bool {
	_, err := os.Stat(s.dbPath)
	return err == nil
}

// GetVersion returns the database build date as a version string
func (s *GeoIPService) GetVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.reader == nil {
		return ""
	}

	md := s.reader.Metadata()
	buildTime := time.Unix(int64(md.BuildEpoch), 0)
	return buildTime.Format("2006-01-02")
}

// LookupIP returns country, province, city for the given IP.
// lang should be "zh" or "en". Returns empty strings if no database or IP not found.
func (s *GeoIPService) LookupIP(ipStr string, lang string) (country, province, city string) {
	s.mu.RLock()
	reader := s.reader
	s.mu.RUnlock()

	if reader == nil {
		return "", "", ""
	}

	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return "", "", ""
	}

	record, err := reader.City(ip)
	if err != nil {
		return "", "", ""
	}

	if lang == "zh" {
		country = record.Country.Names.SimplifiedChinese
		if country == "" {
			country = record.Country.Names.English
		}
		if len(record.Subdivisions) > 0 {
			province = record.Subdivisions[0].Names.SimplifiedChinese
			if province == "" {
				province = record.Subdivisions[0].Names.English
			}
		}
		city = record.City.Names.SimplifiedChinese
		if city == "" {
			city = record.City.Names.English
		}
	} else {
		country = record.Country.Names.English
		if len(record.Subdivisions) > 0 {
			province = record.Subdivisions[0].Names.English
		}
		city = record.City.Names.English
	}

	return country, province, city
}

// IsDownloading returns whether a download is currently in progress
func (s *GeoIPService) IsDownloading() bool {
	s.dlMu.RLock()
	defer s.dlMu.RUnlock()
	return s.downloading
}

// GetDownloadProgress returns the current download progress
func (s *GeoIPService) GetDownloadProgress() DownloadProgress {
	s.dlMu.RLock()
	defer s.dlMu.RUnlock()

	pct := ""
	if s.dlTotal > 0 {
		pct = fmt.Sprintf("%.1f%%", float64(s.dlDownloaded)/float64(s.dlTotal)*100)
	}

	return DownloadProgress{
		Downloading:     s.downloading,
		DownloadedBytes: s.dlDownloaded,
		TotalBytes:      s.dlTotal,
		Percent:         pct,
		Attempt:         s.dlAttempt,
		MaxRetries:      downloadMaxRetries,
		Error:           s.dlError,
	}
}

func (s *GeoIPService) setDownloadProgress(downloaded, total int64, attempt int) {
	s.dlMu.Lock()
	s.dlDownloaded = downloaded
	s.dlTotal = total
	s.dlAttempt = attempt
	s.dlMu.Unlock()
}

// progressReader wraps an io.Reader to track read progress with idle timeout
type progressReader struct {
	reader        io.Reader
	totalBytes    int64
	readBytes     int64
	lastReadTime  time.Time
	idleTimeout   time.Duration
	svc           *GeoIPService
	attempt       int
	lastLoggedPct int // last logged percentage (0, 5, 10, ...)
}

func newProgressReader(r io.Reader, totalBytes int64, svc *GeoIPService, attempt int) *progressReader {
	now := time.Now()
	return &progressReader{
		reader:       r,
		totalBytes:   totalBytes,
		lastReadTime: now,
		idleTimeout:  downloadIdleTimeout,
		svc:          svc,
		attempt:      attempt,
	}
}

func (pr *progressReader) Read(p []byte) (int, error) {
	// Check idle timeout before reading
	if time.Since(pr.lastReadTime) > pr.idleTimeout {
		return 0, fmt.Errorf("download stalled: no data received for %v", pr.idleTimeout)
	}

	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.readBytes += int64(n)
		pr.lastReadTime = time.Now()

		// Update shared progress state
		pr.svc.setDownloadProgress(pr.readBytes, pr.totalBytes, pr.attempt)

		// Log progress every 5% increment
		if pr.totalBytes > 0 {
			currentPct := int(float64(pr.readBytes) / float64(pr.totalBytes) * 100)
			nextLogPct := pr.lastLoggedPct + progressLogStep
			if currentPct >= nextLogPct {
				pr.lastLoggedPct = (currentPct / progressLogStep) * progressLogStep
				slog.Info("GeoIP download progress",
					"downloaded", formatBytes(pr.readBytes),
					"total", formatBytes(pr.totalBytes),
					"percent", fmt.Sprintf("%d%%", pr.lastLoggedPct))
			}
		}
	}
	return n, err
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMG"[exp])
}

// DownloadAndExtract downloads the latest GeoIP database with retry logic.
// Returns nil on success or an error if all retries fail.
// Prevents concurrent downloads.
func (s *GeoIPService) DownloadAndExtract() error {
	// Prevent concurrent downloads
	s.dlMu.Lock()
	if s.downloading {
		s.dlMu.Unlock()
		return fmt.Errorf("download already in progress")
	}
	s.downloading = true
	s.dlDownloaded = 0
	s.dlTotal = 0
	s.dlAttempt = 0
	s.dlError = ""
	s.dlMu.Unlock()

	defer func() {
		s.dlMu.Lock()
		s.downloading = false
		s.dlMu.Unlock()
	}()

	var lastErr error
	for attempt := 1; attempt <= downloadMaxRetries; attempt++ {
		s.setDownloadProgress(0, 0, attempt)
		if attempt > 1 {
			slog.Info("GeoIP download retry", "attempt", attempt, "max", downloadMaxRetries)
		}
		slog.Info("Starting GeoIP database download", "attempt", attempt, "url", geoipDownloadURL)

		err := s.downloadOnce(attempt)
		if err == nil {
			slog.Info("GeoIP database downloaded and extracted successfully")
			// Reload the reader
			s.openReader()
			return nil
		}

		lastErr = err
		slog.Error("GeoIP download attempt failed", "attempt", attempt, "error", err)
	}

	errMsg := fmt.Sprintf("GeoIP download failed after %d attempts: %v", downloadMaxRetries, lastErr)
	s.dlMu.Lock()
	s.dlError = errMsg
	s.dlMu.Unlock()
	return fmt.Errorf("%s", errMsg)
}

func (s *GeoIPService) downloadOnce(attempt int) error {
	// Use a client with no global timeout; we rely on idle timeout instead
	client := &http.Client{
		// Follow redirects (GitHub releases redirect)
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(geoipDownloadURL)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// Wrap body with progress reader for idle timeout detection and logging
	pr := newProgressReader(resp.Body, resp.ContentLength, s, attempt)

	// Extract .mmdb from tar.gz stream
	gzReader, err := gzip.NewReader(pr)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	found := false

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Look for the .mmdb file
		if header.Typeflag == tar.TypeReg && strings.HasSuffix(header.Name, ".mmdb") {
			// Write to a temp file first, then rename for atomicity
			tmpPath := s.dbPath + ".tmp"
			outFile, err := os.Create(tmpPath)
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				os.Remove(tmpPath)
				return fmt.Errorf("failed to write mmdb file: %w", err)
			}

			// Atomic rename
			if err := os.Rename(tmpPath, s.dbPath); err != nil {
				os.Remove(tmpPath)
				return fmt.Errorf("failed to rename temp file: %w", err)
			}

			found = true
			slog.Info("GeoIP database file extracted", "path", s.dbPath)
			break
		}
	}

	if !found {
		return fmt.Errorf("no .mmdb file found in archive")
	}

	return nil
}

// --- Auto-update settings helpers ---

const (
	settingGeoIPAutoUpdate = "geoip_auto_update"
	settingGeoIPUpdateDays = "geoip_update_interval_days"
)

// GetAutoUpdateConfig returns the auto-update settings
func (s *GeoIPService) GetAutoUpdateConfig() (enabled bool, intervalDays int) {
	enabled = false
	intervalDays = 1

	var setting model.SystemSetting
	if err := model.DB.Where("key = ?", settingGeoIPAutoUpdate).First(&setting).Error; err == nil {
		enabled = setting.Value == "true"
	}

	if err := model.DB.Where("key = ?", settingGeoIPUpdateDays).First(&setting).Error; err == nil {
		if v, err := strconv.Atoi(setting.Value); err == nil && v >= 1 && v <= 7 {
			intervalDays = v
		}
	}

	return enabled, intervalDays
}

// SaveAutoUpdateConfig saves the auto-update settings
func (s *GeoIPService) SaveAutoUpdateConfig(enabled bool, intervalDays int) {
	if intervalDays < 1 {
		intervalDays = 1
	}
	if intervalDays > 7 {
		intervalDays = 7
	}

	upsertSetting(settingGeoIPAutoUpdate, fmt.Sprintf("%v", enabled))
	upsertSetting(settingGeoIPUpdateDays, fmt.Sprintf("%d", intervalDays))
}

func upsertSetting(key, value string) {
	var setting model.SystemSetting
	result := model.DB.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		model.DB.Create(&model.SystemSetting{Key: key, Value: value})
	} else {
		model.DB.Model(&setting).Update("value", value)
	}
}
