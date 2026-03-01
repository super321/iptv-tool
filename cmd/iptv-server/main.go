package main

import (
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"iptv-tool-v2/internal/api"
	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/task"
	"iptv-tool-v2/pkg/auth"
	"iptv-tool-v2/web"

	// Import huawei package to trigger init() registration of all EPG strategies
	_ "iptv-tool-v2/internal/iptv/huawei"
)

func main() {
	// Get the absolute path of the executable directory
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	// Command-line flags
	addr := flag.String("addr", ":8080", "HTTP listen address (e.g., :8080 or 0.0.0.0:9090)")
	dataDirFlag := flag.String("data", "data", "Directory for SQLite database (relative to executable by default)")
	logoDirFlag := flag.String("logos", "logos", "Directory for uploaded logo files (relative to executable by default)")
	jwtSecret := flag.String("jwt-secret", "", "JWT secret (auto-generated if empty)")
	flag.Parse()

	// Convert relative paths to absolute paths based on executable location
	dataDir := *dataDirFlag
	if !filepath.IsAbs(dataDir) {
		dataDir = filepath.Join(exeDir, dataDir)
	}

	logoDir := *logoDirFlag
	if !filepath.IsAbs(logoDir) {
		logoDir = filepath.Join(exeDir, logoDir)
	}

	// Ensure directories exist
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(logoDir, 0755)

	// Initialize database
	dbPath := filepath.Join(dataDir, "iptv.db")
	if err := model.InitDB(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize JWT
	auth.InitJWTSecret(*jwtSecret)

	// Initialize and start scheduler
	scheduler := task.NewScheduler()
	if err := scheduler.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// Prepare embedded frontend filesystem
	frontendFS, err := fs.Sub(web.StaticFS, "dist")
	if err != nil {
		log.Fatalf("Failed to load embedded frontend: %v", err)
	}

	// Setup and start HTTP server
	router := api.SetupRouter(scheduler, logoDir, frontendFS)

	log.Printf("IPTV Tool v2 starting on %s", *addr)
	if err := router.Run(*addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
