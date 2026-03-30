package task

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/publish"
	"iptv-tool-v2/internal/service"
)

// --- Schedule configuration type aliases (canonical types in model package) ---

// ScheduleConfig is a type alias for model.ScheduleConfig.
type ScheduleConfig = model.ScheduleConfig

// --- Parsing & Validation ---

// ParseScheduleConfig parses a JSON string into a ScheduleConfig.
// Returns nil if jsonStr is empty.
func ParseScheduleConfig(jsonStr string) (*ScheduleConfig, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return nil, nil
	}
	var cfg ScheduleConfig
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		return nil, fmt.Errorf("invalid schedule config JSON: %w", err)
	}
	return &cfg, nil
}

// MarshalScheduleConfig serializes a ScheduleConfig to JSON string.
// Returns "" if config is nil or empty.
func MarshalScheduleConfig(cfg *ScheduleConfig) string {
	if cfg == nil || cfg.IsEmpty() {
		return ""
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return ""
	}
	return string(data)
}

// ValidateScheduleConfig validates a ScheduleConfig.
func ValidateScheduleConfig(cfg *ScheduleConfig, lang string) error {
	if cfg == nil || cfg.IsEmpty() {
		return nil
	}
	switch cfg.Mode {
	case model.ScheduleModeInterval:
		if cfg.Hours < model.MinIntervalHours || cfg.Hours > model.MaxIntervalHours {
			return fmt.Errorf("error.schedule_invalid_hours")
		}
	case model.ScheduleModeDaily:
		if cfg.Days > model.MaxGeoIPDays {
			return fmt.Errorf("error.schedule_invalid_days")
		}
		if len(cfg.Times) > 0 {
			if err := validateTimePoints(cfg.Times); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("error.invalid_schedule_config")
	}
	return nil
}

// validateTimePoints validates a list of HH:MM time strings.
func validateTimePoints(times []string) error {
	if len(times) == 0 {
		return fmt.Errorf("error.invalid_schedule_config")
	}
	if len(times) > model.MaxTimePoints {
		return fmt.Errorf("error.schedule_max_times")
	}

	// Parse and check duplicates
	seen := make(map[string]bool)
	minutes := make([]int, 0, len(times))
	for _, t := range times {
		t = strings.TrimSpace(t)
		m, err := parseTimeToMinutes(t)
		if err != nil {
			return fmt.Errorf("error.schedule_invalid_time_format")
		}
		if seen[t] {
			return fmt.Errorf("error.schedule_duplicate_time")
		}
		seen[t] = true
		minutes = append(minutes, m)
	}

	// Check minimum gap (circular)
	if len(minutes) > 1 {
		sort.Ints(minutes)
		for i := 1; i < len(minutes); i++ {
			if minutes[i]-minutes[i-1] < model.MinTimeGapMinute {
				return fmt.Errorf("error.schedule_min_interval")
			}
		}
		// Check wrap-around gap (last to first across midnight)
		wrapGap := (1440 - minutes[len(minutes)-1]) + minutes[0]
		if wrapGap < model.MinTimeGapMinute {
			return fmt.Errorf("error.schedule_min_interval")
		}
	}

	return nil
}

// parseTimeToMinutes parses "HH:MM" to minutes since midnight.
func parseTimeToMinutes(t string) (int, error) {
	parts := strings.Split(t, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format")
	}
	var h, m int
	if _, err := fmt.Sscanf(parts[0], "%d", &h); err != nil || h < 0 || h > 23 {
		return 0, fmt.Errorf("invalid hour")
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &m); err != nil || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid minute")
	}
	return h*60 + m, nil
}

// --- Scheduler ---

// Scheduler manages all scheduled tasks using gocron/v2.
type Scheduler struct {
	liveService   *service.LiveSourceService
	epgService    *service.EPGSourceService
	detectService *service.DetectService

	cron gocron.Scheduler
	mu   sync.Mutex
	jobs map[string]uuid.UUID // tag -> job UUID

	accessStatSvc *service.AccessStatService
	geoipSvc      *service.GeoIPService
}

// NewScheduler creates a new task scheduler.
func NewScheduler(dataDir string) *Scheduler {
	cronScheduler, err := gocron.NewScheduler(
		gocron.WithLimitConcurrentJobs(10, gocron.LimitModeWait),
	)
	if err != nil {
		slog.Error("Failed to create gocron scheduler", "error", err)
		panic(fmt.Sprintf("failed to create gocron scheduler: %v", err))
	}
	return &Scheduler{
		liveService:   service.NewLiveSourceService(),
		epgService:    service.NewEPGSourceService(),
		detectService: service.NewDetectService(dataDir),
		cron:          cronScheduler,
		jobs:          make(map[string]uuid.UUID),
	}
}

// SetAccessStatService sets the access stat service for cleanup tasks.
func (s *Scheduler) SetAccessStatService(svc *service.AccessStatService) {
	s.accessStatSvc = svc
}

// SetGeoIPService sets the GeoIP service for auto-update tasks.
func (s *Scheduler) SetGeoIPService(svc *service.GeoIPService) {
	s.geoipSvc = svc
}

// Start initializes and starts all scheduled tasks from the database.
func (s *Scheduler) Start() error {
	slog.Info("Initializing task scheduler...")

	// Load all enabled live sources and register their scheduled tasks
	var liveSources []model.LiveSource
	if err := model.DB.Where("status = ?", true).Find(&liveSources).Error; err != nil {
		return fmt.Errorf("failed to load live sources: %w", err)
	}

	for _, src := range liveSources {
		if src.Type == model.LiveSourceTypeNetworkManual {
			continue
		}
		if src.CronTime == "" {
			continue
		}
		cfg, err := ParseScheduleConfig(src.CronTime)
		if err != nil {
			slog.Warn("Failed to parse schedule config for live source", "name", src.Name, "id", src.ID, "error", err)
			continue
		}
		if err := s.AddLiveSourceTask(src.ID, cfg); err != nil {
			slog.Warn("Failed to schedule live source", "name", src.Name, "id", src.ID, "error", err)
		}
	}

	// Load all enabled EPG sources and register their scheduled tasks
	var epgSources []model.EPGSource
	if err := model.DB.Where("status = ?", true).Find(&epgSources).Error; err != nil {
		return fmt.Errorf("failed to load EPG sources: %w", err)
	}

	for _, src := range epgSources {
		if src.CronTime == "" {
			continue
		}
		cfg, err := ParseScheduleConfig(src.CronTime)
		if err != nil {
			slog.Warn("Failed to parse schedule config for EPG source", "name", src.Name, "id", src.ID, "error", err)
			continue
		}
		if err := s.AddEPGSourceTask(src.ID, cfg); err != nil {
			slog.Warn("Failed to schedule EPG source", "name", src.Name, "id", src.ID, "error", err)
		}
	}

	// Load all enabled live sources with detect config
	var detectSources []model.LiveSource
	if err := model.DB.Where("status = ? AND cron_detect != ''", true).Find(&detectSources).Error; err != nil {
		return fmt.Errorf("failed to load live sources for detection: %w", err)
	}

	for _, src := range detectSources {
		cfg, err := ParseScheduleConfig(src.CronDetect)
		if err != nil {
			slog.Warn("Failed to parse detect schedule config", "name", src.Name, "id", src.ID, "error", err)
			continue
		}
		if err := s.AddDetectTask(src.ID, cfg, src.DetectStrategy); err != nil {
			slog.Warn("Failed to schedule detect task", "name", src.Name, "id", src.ID, "error", err)
		}
	}

	// Start GeoIP auto-update task if enabled
	if s.geoipSvc != nil {
		enabled, cfg := s.geoipSvc.GetAutoUpdateConfig()
		if enabled {
			s.AddGeoIPUpdateTask(&cfg)
		}
	}

	// Start access stats cleanup task (daily at 03:00)
	s.startCleanupTask()

	// Start the gocron scheduler
	s.cron.Start()

	s.mu.Lock()
	jobCount := len(s.jobs)
	s.mu.Unlock()
	slog.Info("Task scheduler started", "total_jobs", jobCount)
	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	if err := s.cron.Shutdown(); err != nil {
		slog.Error("Failed to shutdown gocron scheduler", "error", err)
	}
	slog.Info("Task scheduler stopped.")
}

// --- Job helpers ---

// removeJobByTag removes an existing job by its tag. Caller must NOT hold s.mu.
func (s *Scheduler) removeJobByTag(tag string) {
	s.mu.Lock()
	jobID, exists := s.jobs[tag]
	if exists {
		delete(s.jobs, tag)
	}
	s.mu.Unlock()

	if exists {
		if err := s.cron.RemoveJob(jobID); err != nil {
			slog.Warn("Failed to remove job", "tag", tag, "error", err)
		}
	}
}

// addJob creates a gocron job from a ScheduleConfig. Returns an error if config is invalid.
func (s *Scheduler) addJob(tag string, cfg *ScheduleConfig, taskFunc func()) error {
	if cfg == nil || cfg.IsEmpty() {
		return nil
	}

	// Remove existing job with same tag first
	s.removeJobByTag(tag)

	var jobDef gocron.JobDefinition

	switch cfg.Mode {
	case model.ScheduleModeInterval:
		jobDef = gocron.DurationJob(time.Duration(cfg.Hours) * time.Hour)
	case model.ScheduleModeDaily:
		days := cfg.Days
		if days < 1 {
			days = 1
		}
		if len(cfg.Times) > 0 {
			atTimes, err := buildAtTimes(cfg.Times)
			if err != nil {
				return err
			}
			jobDef = gocron.DailyJob(uint(days), atTimes)
		} else {
			// No specific times: use duration-based interval (days * 24h)
			jobDef = gocron.DurationJob(time.Duration(days) * 24 * time.Hour)
		}
	default:
		return fmt.Errorf("unsupported schedule mode: %s", cfg.Mode)
	}

	// Wrap taskFunc with panic recovery
	safeFunc := func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Scheduler task panicked", "tag", tag, "panic", r)
			}
		}()
		taskFunc()
	}

	job, err := s.cron.NewJob(
		jobDef,
		gocron.NewTask(safeFunc),
		gocron.WithTags(tag),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return fmt.Errorf("failed to create job [%s]: %w", tag, err)
	}

	s.mu.Lock()
	s.jobs[tag] = job.ID()
	s.mu.Unlock()

	return nil
}

// buildAtTimes converts []string{"HH:MM",...} to gocron.AtTimes.
func buildAtTimes(times []string) (gocron.AtTimes, error) {
	atTimeList := make([]gocron.AtTime, 0, len(times))
	for _, t := range times {
		m, err := parseTimeToMinutes(t)
		if err != nil {
			return nil, fmt.Errorf("invalid time: %s", t)
		}
		h := m / 60
		min := m % 60
		atTimeList = append(atTimeList, gocron.NewAtTime(uint(h), uint(min), 0))
	}
	return gocron.NewAtTimes(atTimeList[0], atTimeList[1:]...), nil
}

// --- Task tag generators ---

func liveSourceTag(sourceID uint) string { return fmt.Sprintf("live_sync_%d", sourceID) }
func epgSourceTag(sourceID uint) string  { return fmt.Sprintf("epg_sync_%d", sourceID) }
func detectTag(sourceID uint) string     { return fmt.Sprintf("detect_%d", sourceID) }
func geoipTag() string                   { return "geoip_update" }
func cleanupTag() string                 { return "access_stats_cleanup" }

// --- Public API ---

// AddLiveSourceTask adds or updates a scheduled task for a live source.
func (s *Scheduler) AddLiveSourceTask(sourceID uint, cfg *ScheduleConfig) error {
	if cfg == nil || cfg.IsEmpty() {
		return nil
	}
	tag := liveSourceTag(sourceID)
	id := sourceID
	err := s.addJob(tag, cfg, func() {
		slog.Info("Scheduler: fetching live source", "id", id)
		if err := s.liveService.FetchAndUpdate(id); err != nil {
			slog.Error("Scheduler: failed to fetch live source", "id", id, "error", err)
		} else {
			publish.InvalidateAll()
		}
	})
	if err != nil {
		return err
	}
	slog.Info("Scheduled live source task", "id", sourceID, "config", cfg.String())
	return nil
}

// RemoveLiveSourceTask removes a scheduled task for a live source.
func (s *Scheduler) RemoveLiveSourceTask(sourceID uint) {
	tag := liveSourceTag(sourceID)
	s.removeJobByTag(tag)
	slog.Info("Removed live source task", "id", sourceID)
}

// AddEPGSourceTask adds or updates a scheduled task for an EPG source.
func (s *Scheduler) AddEPGSourceTask(sourceID uint, cfg *ScheduleConfig) error {
	if cfg == nil || cfg.IsEmpty() {
		return nil
	}
	tag := epgSourceTag(sourceID)
	id := sourceID
	err := s.addJob(tag, cfg, func() {
		slog.Info("Scheduler: fetching EPG source", "id", id)
		if err := s.epgService.FetchAndUpdate(id); err != nil {
			slog.Error("Scheduler: failed to fetch EPG source", "id", id, "error", err)
		} else {
			publish.InvalidateAll()
		}
	})
	if err != nil {
		return err
	}
	slog.Info("Scheduled EPG source task", "id", sourceID, "config", cfg.String())
	return nil
}

// RemoveEPGSourceTask removes a scheduled task for an EPG source.
func (s *Scheduler) RemoveEPGSourceTask(sourceID uint) {
	tag := epgSourceTag(sourceID)
	s.removeJobByTag(tag)
	slog.Info("Removed EPG source task", "id", sourceID)
}

// TriggerLiveSourceNow manually triggers a live source fetch immediately.
func (s *Scheduler) TriggerLiveSourceNow(sourceID uint) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Manual trigger panicked", "type", "live_source", "id", sourceID, "panic", r)
			}
		}()
		slog.Info("Manual trigger: fetching live source", "id", sourceID)
		if err := s.liveService.FetchAndUpdate(sourceID); err != nil {
			slog.Error("Manual trigger: failed to fetch live source", "id", sourceID, "error", err)
		} else {
			publish.InvalidateAll()
		}
	}()
}

// TriggerEPGSourceNow manually triggers an EPG source fetch immediately.
func (s *Scheduler) TriggerEPGSourceNow(sourceID uint) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Manual trigger panicked", "type", "epg_source", "id", sourceID, "panic", r)
			}
		}()
		slog.Info("Manual trigger: fetching EPG source", "id", sourceID)
		if err := s.epgService.FetchAndUpdate(sourceID); err != nil {
			slog.Error("Manual trigger: failed to fetch EPG source", "id", sourceID, "error", err)
		} else {
			publish.InvalidateAll()
		}
	}()
}

// AddDetectTask adds or updates a scheduled task for channel detection.
func (s *Scheduler) AddDetectTask(sourceID uint, cfg *ScheduleConfig, strategy string) error {
	if cfg == nil || cfg.IsEmpty() {
		return nil
	}
	tag := detectTag(sourceID)
	id := sourceID
	st := strategy
	err := s.addJob(tag, cfg, func() {
		slog.Info("Scheduler: detecting channels for live source", "id", id, "strategy", st)
		if err := s.detectService.DetectChannels(id, false, st); err != nil {
			slog.Error("Scheduler: failed to detect channels", "id", id, "error", err)
		} else {
			publish.InvalidateAll()
		}
	})
	if err != nil {
		return err
	}
	slog.Info("Scheduled detect task", "id", sourceID, "strategy", strategy, "config", cfg.String())
	return nil
}

// RemoveDetectTask removes a scheduled task for channel detection.
func (s *Scheduler) RemoveDetectTask(sourceID uint) {
	tag := detectTag(sourceID)
	s.removeJobByTag(tag)
	slog.Info("Removed detect task", "id", sourceID)
}

// CheckFFprobe checks whether the ffprobe executable is available.
func (s *Scheduler) CheckFFprobe() error {
	_, _, err := s.detectService.GetFFprobePath()
	return err
}

// TriggerDetectNow manually triggers channel detection immediately.
func (s *Scheduler) TriggerDetectNow(sourceID uint, strategy string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Manual trigger panicked", "type", "detect", "id", sourceID, "panic", r)
			}
		}()
		slog.Info("Manual trigger: detecting channels for live source", "id", sourceID, "strategy", strategy)
		if err := s.detectService.DetectChannels(sourceID, true, strategy); err != nil {
			slog.Error("Manual trigger: failed to detect channels", "id", sourceID, "error", err)
		} else {
			publish.InvalidateAll()
		}
	}()
}

// --- GeoIP auto-update task ---

// AddGeoIPUpdateTask starts a periodic GeoIP database update task.
func (s *Scheduler) AddGeoIPUpdateTask(cfg *ScheduleConfig) {
	if s.geoipSvc == nil || cfg == nil || cfg.IsEmpty() {
		return
	}
	tag := geoipTag()
	err := s.addJob(tag, cfg, func() {
		slog.Info("Scheduler: auto-updating GeoIP database")
		if err := s.geoipSvc.DownloadAndExtract(); err != nil {
			slog.Error("Scheduler: failed to auto-update GeoIP database", "error", err)
		}
	})
	if err != nil {
		slog.Error("Failed to schedule GeoIP update task", "error", err)
		return
	}
	slog.Info("Scheduled GeoIP auto-update task", "config", cfg.String())
}

// RemoveGeoIPUpdateTask stops the GeoIP auto-update task.
func (s *Scheduler) RemoveGeoIPUpdateTask() {
	s.removeJobByTag(geoipTag())
	slog.Info("Removed GeoIP auto-update task")
}

// --- Access stats cleanup task ---

func (s *Scheduler) startCleanupTask() {
	tag := cleanupTag()
	safeFunc := func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Cleanup task panicked", "panic", r)
			}
		}()
		if s.accessStatSvc != nil {
			s.accessStatSvc.Cleanup()
		}
	}

	job, err := s.cron.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(3, 0, 0))),
		gocron.NewTask(safeFunc),
		gocron.WithTags(tag),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		slog.Error("Failed to schedule cleanup task", "error", err)
		return
	}

	s.mu.Lock()
	s.jobs[tag] = job.ID()
	s.mu.Unlock()

	slog.Info("Scheduled access stats cleanup task (daily at 03:00)")
}
