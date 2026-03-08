package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"iptv-tool-v2/internal/model"
)

const (
	DefaultDetectConcurrency = 10
	DefaultDetectTimeout     = 5
)

// DetectService handles channel detection using ffprobe
type DetectService struct {
	dataDir string
}

func NewDetectService(dataDir string) *DetectService {
	return &DetectService{dataDir: dataDir}
}

// GetFFprobePath returns the path to the ffprobe executable, or error if not found
func (s *DetectService) GetFFprobePath() (string, error) {
	name := "ffprobe"
	if runtime.GOOS == "windows" {
		name = "ffprobe.exe"
	}
	ffprobePath := filepath.Join(s.dataDir, "detect", name)

	// Check existence by trying to run
	cmd := exec.Command(ffprobePath, "-version")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffprobe 可执行文件未找到或无法运行: %w", err)
	}
	return ffprobePath, nil
}

// GetFFprobeVersion returns the version string of the installed ffprobe
func (s *DetectService) GetFFprobeVersion() (string, error) {
	ffprobePath, err := s.GetFFprobePath()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobePath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("获取 ffprobe 版本失败: %w", err)
	}

	// Parse first line: "ffprobe version N-xxxxx-gxxxxxxx Copyright ..."
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	return "unknown", nil
}

// getDetectConfig reads concurrency and timeout settings from the database
func (s *DetectService) getDetectConfig() (concurrency int, timeout int) {
	concurrency = DefaultDetectConcurrency
	timeout = DefaultDetectTimeout

	var settings []model.SystemSetting
	model.DB.Where("key IN ?", []string{"detect_concurrency", "detect_timeout"}).Find(&settings)

	for _, setting := range settings {
		switch setting.Key {
		case "detect_concurrency":
			if v, err := strconv.Atoi(setting.Value); err == nil && v >= 1 && v <= 30 {
				concurrency = v
			}
		case "detect_timeout":
			if v, err := strconv.Atoi(setting.Value); err == nil && v >= 1 && v <= 30 {
				timeout = v
			}
		}
	}
	return
}

// DetectChannels performs detection on all parsed channels for a given source.
// manual=true: fails immediately if the source is syncing.
// manual=false: waits for syncing to finish (up to 10 minutes) before starting detection.
func (s *DetectService) DetectChannels(sourceID uint, manual bool) error {
	var source model.LiveSource
	if err := model.DB.First(&source, sourceID).Error; err != nil {
		return fmt.Errorf("直播源 %d 未找到: %w", sourceID, err)
	}

	if !source.Status {
		return nil // Source is disabled, skip
	}

	// Check if already detecting
	if source.IsDetecting {
		return fmt.Errorf("该直播源正在检测中，请勿重复触发")
	}

	// Check syncing status
	if manual {
		if source.IsSyncing {
			return fmt.Errorf("该直播源正在刷新中，请等待刷新完成后再执行检测")
		}
	} else {
		// Wait for syncing to finish (auto/scheduled mode)
		if err := s.waitForSyncComplete(sourceID, 10*time.Minute); err != nil {
			return err
		}
	}

	// Get ffprobe path
	ffprobePath, err := s.GetFFprobePath()
	if err != nil {
		return err
	}

	// Mark as detecting
	model.DB.Model(&model.LiveSource{}).Where("id = ?", sourceID).Update("is_detecting", true)
	defer func() {
		model.DB.Model(&model.LiveSource{}).Where("id = ?", sourceID).Update("is_detecting", false)
	}()

	// Reset latency, video_codec, video_resolution, and detected_at for all channels in this source before detecting
	model.DB.Model(&model.ParsedChannel{}).Where("source_id = ?", sourceID).Updates(map[string]interface{}{
		"latency":          nil,
		"detected_at":      nil,
		"video_codec":      nil,
		"video_resolution": nil,
	})

	// Load channels
	var channels []model.ParsedChannel
	if err := model.DB.Where("source_id = ?", sourceID).Find(&channels).Error; err != nil {
		return fmt.Errorf("加载频道列表失败: %w", err)
	}

	if len(channels) == 0 {
		return nil
	}

	concurrency, timeout := s.getDetectConfig()

	slog.Info("Starting channel detection",
		"source_id", sourceID,
		"channels", len(channels),
		"concurrency", concurrency,
		"timeout_seconds", timeout,
	)

	// Concurrent detection with semaphore
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := range channels {
		wg.Add(1)
		go func(ch *model.ParsedChannel) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			// For channels with multiple URLs (pipe-separated), test the first one
			testURL := ch.URL
			if idx := strings.Index(testURL, "|"); idx > 0 {
				testURL = strings.TrimSpace(testURL[:idx])
			}

			latency, codec, resolution, detectErr := s.detectSingleChannel(ffprobePath, testURL, timeout)
			now := time.Now()

			if detectErr != nil {
				// Timeout or error
				timeoutVal := -1
				ch.Latency = &timeoutVal
			} else {
				ch.Latency = &latency
				if codec != "" {
					ch.VideoCodec = &codec
				}
				if resolution != "" {
					ch.VideoResolution = &resolution
				}
			}
			ch.DetectedAt = &now

			// Update single channel result in DB
			model.DB.Model(&model.ParsedChannel{}).Where("id = ?", ch.ID).Updates(map[string]interface{}{
				"latency":          ch.Latency,
				"detected_at":      ch.DetectedAt,
				"video_codec":      ch.VideoCodec,
				"video_resolution": ch.VideoResolution,
			})
		}(&channels[i])
	}

	wg.Wait()

	slog.Info("Channel detection completed", "source_id", sourceID, "channels", len(channels))
	return nil
}

// ffprobeResult represents the JSON output from ffprobe -show_streams
type ffprobeResult struct {
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"` // "video", "audio"
	CodecName string `json:"codec_name"` // "h264", "hevc", etc.
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// detectSingleChannel runs ffprobe to probe a single URL and returns the latency, video codec, and resolution
func (s *DetectService) detectSingleChannel(ffprobePath string, url string, timeoutSec int) (latency int, codec string, resolution string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	start := time.Now()

	// ffprobe -v quiet -print_format json -show_streams -i <url>
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-i", url,
	)

	output, runErr := cmd.Output()

	elapsed := time.Since(start)
	latencyMs := int(elapsed.Milliseconds())

	if ctx.Err() == context.DeadlineExceeded {
		return 0, "", "", fmt.Errorf("timeout after %ds", timeoutSec)
	}

	if runErr != nil {
		// ffprobe returned non-zero exit code — stream is unreachable or invalid
		return 0, "", "", fmt.Errorf("ffprobe error: %w", runErr)
	}

	// Parse JSON output
	var result ffprobeResult
	if err := json.Unmarshal(output, &result); err != nil {
		// ffprobe ran successfully but JSON parsing failed — still return latency
		return latencyMs, "", "", nil
	}

	// Find the first video stream
	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			codec = stream.CodecName
			if stream.Width > 0 && stream.Height > 0 {
				resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)
			}
			break
		}
	}

	return latencyMs, codec, resolution, nil
}

// waitForSyncComplete polls the source's is_syncing status until it becomes false
func (s *DetectService) waitForSyncComplete(sourceID uint, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		var source model.LiveSource
		if err := model.DB.First(&source, sourceID).Error; err != nil {
			return fmt.Errorf("直播源 %d 未找到: %w", sourceID, err)
		}
		if !source.IsSyncing {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("等待直播源刷新完成超时（超过 %v）", maxWait)
		}
		<-ticker.C
	}
}
