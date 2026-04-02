package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"iptv-tool-v2/internal/model"
)

const (
	DefaultHTTPSPort = 8024
	CertDir          = "certs"
	ServerCertFile   = "server.crt"
	ServerKeyFile    = "server.key"
	CACertFile       = "ca.crt"
)

// HTTPSConfig holds the HTTPS configuration loaded from SystemSetting
type HTTPSConfig struct {
	Enabled   bool
	Port      int
	MutualTLS bool
}

// HTTPSService manages the HTTPS server lifecycle
type HTTPSService struct {
	dataDir  string
	handler  http.Handler
	httpAddr string // HTTP listen address for conflict detection

	mu     sync.Mutex
	server *http.Server
}

// NewHTTPSService creates a new HTTPS service manager
func NewHTTPSService(dataDir string, handler http.Handler, httpAddr string) *HTTPSService {
	return &HTTPSService{
		dataDir:  dataDir,
		handler:  handler,
		httpAddr: httpAddr,
	}
}

// SetHandler sets the HTTP handler for the HTTPS server (used for two-phase initialization)
func (s *HTTPSService) SetHandler(handler http.Handler) {
	s.handler = handler
}

// HTTPPort returns the HTTP listen port extracted from the httpAddr (e.g. ":8023" → 8023)
func (s *HTTPSService) HTTPPort() int {
	addr := s.httpAddr
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		if port, err := strconv.Atoi(addr[idx+1:]); err == nil {
			return port
		}
	}
	return 0
}

// CertsDir returns the path to the certificate directory
func (s *HTTPSService) CertsDir() string {
	return filepath.Join(s.dataDir, CertDir)
}

// CertPath returns the path to the server certificate
func (s *HTTPSService) CertPath() string {
	return filepath.Join(s.CertsDir(), ServerCertFile)
}

// KeyPath returns the path to the server private key
func (s *HTTPSService) KeyPath() string {
	return filepath.Join(s.CertsDir(), ServerKeyFile)
}

// CACertPath returns the path to the CA certificate
func (s *HTTPSService) CACertPath() string {
	return filepath.Join(s.CertsDir(), CACertFile)
}

// HasCert checks if the server certificate file exists
func (s *HTTPSService) HasCert() bool {
	_, err := os.Stat(s.CertPath())
	return err == nil
}

// HasKey checks if the server private key file exists
func (s *HTTPSService) HasKey() bool {
	_, err := os.Stat(s.KeyPath())
	return err == nil
}

// HasCACert checks if the CA certificate file exists
func (s *HTTPSService) HasCACert() bool {
	_, err := os.Stat(s.CACertPath())
	return err == nil
}

// LoadConfig reads HTTPS configuration from the database
func (s *HTTPSService) LoadConfig() HTTPSConfig {
	cfg := HTTPSConfig{
		Port: DefaultHTTPSPort,
	}

	var settings []model.SystemSetting
	model.DB.Where("key IN ?", []string{"https_enabled", "https_port", "https_mutual_tls"}).Find(&settings)

	for _, setting := range settings {
		switch setting.Key {
		case "https_enabled":
			cfg.Enabled = setting.Value == "true"
		case "https_port":
			if port, err := strconv.Atoi(setting.Value); err == nil {
				cfg.Port = port
			}
		case "https_mutual_tls":
			cfg.MutualTLS = setting.Value == "true"
		}
	}

	return cfg
}

// SaveConfig persists HTTPS configuration to the database
func (s *HTTPSService) SaveConfig(cfg HTTPSConfig) {
	upsertSetting("https_enabled", strconv.FormatBool(cfg.Enabled))
	upsertSetting("https_port", strconv.Itoa(cfg.Port))
	upsertSetting("https_mutual_tls", strconv.FormatBool(cfg.MutualTLS))
}

// ValidateCertKeyPair checks if the cert and key on disk form a valid TLS pair
func (s *HTTPSService) ValidateCertKeyPair() error {
	_, err := tls.LoadX509KeyPair(s.CertPath(), s.KeyPath())
	return err
}

// LoadAndStart loads configuration from DB and starts HTTPS if enabled
func (s *HTTPSService) LoadAndStart() {
	cfg := s.LoadConfig()
	if !cfg.Enabled {
		slog.Info("HTTPS is disabled, skipping startup")
		return
	}

	if !s.HasCert() || !s.HasKey() {
		slog.Warn("HTTPS is enabled but certificate or key file is missing, skipping startup")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.startServerLocked(cfg); err != nil {
		slog.Error("Failed to start HTTPS server", "error", err)
	}
}

// Restart stops the current HTTPS server (if any) and starts a new one with current config.
// The entire stop→start sequence is atomic under the mutex to prevent interleaving.
// Returns nil if HTTPS is disabled (stops cleanly).
func (s *HTTPSService) Restart() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing server if any
	s.stopServerLocked()

	cfg := s.LoadConfig()
	if !cfg.Enabled {
		slog.Info("HTTPS disabled, server stopped")
		return nil
	}

	if !s.HasCert() || !s.HasKey() {
		return fmt.Errorf("certificate or key file missing")
	}

	return s.startServerLocked(cfg)
}

// Stop gracefully shuts down the HTTPS server
func (s *HTTPSService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopServerLocked()
}

// startServerLocked creates and starts the HTTPS server.
// MUST be called with s.mu held.
func (s *HTTPSService) startServerLocked(cfg HTTPSConfig) error {
	// Load certificate
	cert, err := tls.LoadX509KeyPair(s.CertPath(), s.KeyPath())
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Configure mutual TLS if enabled
	if cfg.MutualTLS && s.HasCACert() {
		caCert, err := os.ReadFile(s.CACertPath())
		if err != nil {
			return fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = caCertPool
	}

	addr := fmt.Sprintf(":%d", cfg.Port)

	// Bind the port synchronously so errors (e.g. port conflict) are returned immediately.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	tlsListener := tls.NewListener(ln, tlsConfig)

	s.server = &http.Server{
		Handler:   s.handler,
		TLSConfig: tlsConfig,
	}

	go func() {
		slog.Info("HTTPS server started", "address", addr, "mutual_tls", cfg.MutualTLS)
		if err := s.server.Serve(tlsListener); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTPS server error", "error", err)
		}
	}()

	return nil
}

// stopServerLocked gracefully shuts down the current HTTPS server.
// MUST be called with s.mu held.
func (s *HTTPSService) stopServerLocked() {
	if s.server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slog.Info("Shutting down HTTPS server")
	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error("HTTPS server shutdown error", "error", err)
	}
	s.server = nil
}

// Note: upsertSetting is defined in geoip.go within the same package
