package version

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X iptv-tool-v2/internal/version.Version=v1.0.0"
var Version = "dev"
