package main

import (
	"flag"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"iptv-tool-v2/internal/api"
	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/internal/task"
	"iptv-tool-v2/locales"
	"iptv-tool-v2/pkg/auth"
	"iptv-tool-v2/pkg/i18n"
	"iptv-tool-v2/pkg/logger"
	"iptv-tool-v2/web"

	// Import huawei package to trigger init() registration of all EPG strategies
	_ "iptv-tool-v2/internal/iptv/huawei"
)

func main() {
	// Get the absolute path of the executable directory
	exePath, err := os.Executable()
	if err != nil {
		slog.Error("Failed to get executable path", "error", err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exePath)

	// Command-line flags
	addr := flag.String("addr", ":8023", "HTTP listen address (e.g., :8023 or 0.0.0.0:9090)")
	dataDirFlag := flag.String("data", "data", "Directory for data storage including db and logos (relative to executable by default)")
	logDirFlag := flag.String("log-dir", "logs", "Directory for log files (relative to executable by default)")
	jwtSecret := flag.String("jwt-secret", "", "JWT secret (auto-generated if empty)")
	flag.Parse()

	// Convert relative paths to absolute paths based on executable location
	dataDir := *dataDirFlag
	if !filepath.IsAbs(dataDir) {
		dataDir = filepath.Join(exeDir, dataDir)
	}
	logDir := *logDirFlag
	if !filepath.IsAbs(logDir) {
		logDir = filepath.Join(exeDir, logDir)
	}

	// Create log buffers for the web UI log center
	runtimeLogBuf := api.NewRuntimeLogBuffer(10000)
	accessLogBuf := api.NewAccessLogBuffer(10000)

	// Initialize logger early, with runtime log buffer tee
	if err := logger.InitLogger(logDir, runtimeLogBuf); err != nil {
		// Fallback to basic logging if logger init fails
		slog.Error("Failed to initialize logger", "error", err)
		os.Exit(1)
	}

	// Define subdirectories
	dbDir := filepath.Join(dataDir, "db")
	logoDir := filepath.Join(dataDir, "logos")
	detectDir := filepath.Join(dataDir, "detect")

	// Ensure directories exist
	os.MkdirAll(dbDir, 0755)
	os.MkdirAll(logoDir, 0755)
	os.MkdirAll(detectDir, 0755)

	// Initialize database
	dbPath := filepath.Join(dbDir, "iptv.db")
	if err := model.InitDB(dbPath); err != nil {
		logger.Fatalf("Failed to initialize database", "error", err)
	}

	// Initialize JWT
	auth.InitJWTSecret(*jwtSecret)

	// Initialize i18n
	i18n.Init(locales.FS)

	// Initialize GeoIP service
	geoipSvc := service.NewGeoIPService(dataDir)
	defer geoipSvc.Close()

	// Initialize AccessStat service (starts background worker)
	accessStatSvc := service.NewAccessStatService(geoipSvc)
	defer accessStatSvc.Stop()

	// Initialize and start scheduler
	scheduler := task.NewScheduler(dataDir)
	scheduler.SetGeoIPService(geoipSvc)
	scheduler.SetAccessStatService(accessStatSvc)
	if err := scheduler.Start(); err != nil {
		logger.Fatalf("Failed to start scheduler", "error", err)
	}
	defer scheduler.Stop()

	// Prepare embedded frontend filesystem
	frontendFS, err := fs.Sub(web.StaticFS, "dist")
	if err != nil {
		logger.Fatalf("Failed to load embedded frontend", "error", err)
	}

	// Setup and start HTTP server
	router := api.SetupRouter(scheduler, logoDir, dataDir, frontendFS, runtimeLogBuf, accessLogBuf, geoipSvc, accessStatSvc)

	slog.Info("IPTV Tool v2 starting", "address", *addr)
	if err := router.Run(*addr); err != nil {
		logger.Fatalf("Failed to start server", "error", err)
	}
}
