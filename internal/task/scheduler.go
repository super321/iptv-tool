package task

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/service"
)

// validCronTimes is the set of valid user-facing interval keys.
var validCronTimes = map[string]bool{
	"1h": true, "2h": true, "4h": true,
	"6h": true, "12h": true, "24h": true,
}

// BuildCronExpr generates a cron expression for the given interval key based on
// the current time, so that tasks are spread across the clock instead of all
// firing at the top of every hour.
//
// Examples (assuming current time is 16:19):
//
//	"1h"  -> "19 * * * *"       (every hour at minute 19)
//	"2h"  -> "19 */2 * * *"     (every 2 hours at minute 19)
//	"24h" -> "19 16 * * *"      (every day at 16:19)
func BuildCronExpr(cronTime string) (string, error) {
	if !validCronTimes[cronTime] {
		return "", fmt.Errorf("invalid cron time: %s", cronTime)
	}

	now := time.Now()
	minute := now.Minute()
	hour := now.Hour()

	switch cronTime {
	case "1h":
		return fmt.Sprintf("%d * * * *", minute), nil
	case "24h":
		return fmt.Sprintf("%d %d * * *", minute, hour), nil
	default:
		// 2h, 4h, 6h, 12h — extract the number portion
		intervalStr := cronTime[:len(cronTime)-1] // strip trailing "h"
		return fmt.Sprintf("%d */%s * * *", minute, intervalStr), nil
	}
}

// CronTimeOptions returns the available options for the frontend dropdown.
// Labels are i18n keys that should be translated at the handler level.
var CronTimeOptions = []map[string]string{
	{"value": "1h", "label": "label.cron_1h"},
	{"value": "2h", "label": "label.cron_2h"},
	{"value": "4h", "label": "label.cron_4h"},
	{"value": "6h", "label": "label.cron_6h"},
	{"value": "12h", "label": "label.cron_12h"},
	{"value": "24h", "label": "label.cron_24h"},
}

// Scheduler manages all cron jobs for live sources and EPG sources
type Scheduler struct {
	cron          *cron.Cron
	liveService   *service.LiveSourceService
	epgService    *service.EPGSourceService
	detectService *service.DetectService

	mu            sync.Mutex
	liveEntries   map[uint]cron.EntryID // sourceID -> cron entry ID
	epgEntries    map[uint]cron.EntryID // sourceID -> cron entry ID
	detectEntries map[uint]cron.EntryID // sourceID -> detect cron entry ID
}

// NewScheduler creates a new task scheduler
func NewScheduler(dataDir string) *Scheduler {
	return &Scheduler{
		cron:          cron.New(),
		liveService:   service.NewLiveSourceService(),
		epgService:    service.NewEPGSourceService(),
		detectService: service.NewDetectService(dataDir),
		liveEntries:   make(map[uint]cron.EntryID),
		epgEntries:    make(map[uint]cron.EntryID),
		detectEntries: make(map[uint]cron.EntryID),
	}
}

// Start initializes and starts all scheduled tasks from the database
func (s *Scheduler) Start() error {
	slog.Info("Initializing task scheduler...")

	// Load all enabled live sources and register their cron jobs
	var liveSources []model.LiveSource
	if err := model.DB.Where("status = ?", true).Find(&liveSources).Error; err != nil {
		return fmt.Errorf("failed to load live sources: %w", err)
	}

	for _, src := range liveSources {
		// network_manual sources are static content, no need for periodic fetching
		if src.Type == model.LiveSourceTypeNetworkManual {
			continue
		}
		if src.CronTime == "" {
			continue
		}
		if err := s.AddLiveSourceTask(src.ID, src.CronTime); err != nil {
			slog.Warn("Failed to schedule live source", "name", src.Name, "id", src.ID, "error", err)
		}
	}

	// Load all enabled EPG sources and register their cron jobs
	var epgSources []model.EPGSource
	if err := model.DB.Where("status = ?", true).Find(&epgSources).Error; err != nil {
		return fmt.Errorf("failed to load EPG sources: %w", err)
	}

	for _, src := range epgSources {
		if src.CronTime == "" {
			continue
		}
		if err := s.AddEPGSourceTask(src.ID, src.CronTime); err != nil {
			slog.Warn("Failed to schedule EPG source", "name", src.Name, "id", src.ID, "error", err)
		}
	}

	// Load all enabled live sources with cron_detect and register their detect cron jobs
	var detectSources []model.LiveSource
	if err := model.DB.Where("status = ? AND cron_detect != ''", true).Find(&detectSources).Error; err != nil {
		return fmt.Errorf("failed to load live sources for detection: %w", err)
	}

	for _, src := range detectSources {
		if err := s.AddDetectTask(src.ID, src.CronDetect, src.DetectStrategy); err != nil {
			slog.Warn("Failed to schedule detect task", "name", src.Name, "id", src.ID, "error", err)
		}
	}

	s.cron.Start()
	slog.Info("Task scheduler started", "live_tasks", len(s.liveEntries), "epg_tasks", len(s.epgEntries), "detect_tasks", len(s.detectEntries))
	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	slog.Info("Task scheduler stopped.")
}

// AddLiveSourceTask adds or updates a cron job for a live source
func (s *Scheduler) AddLiveSourceTask(sourceID uint, cronTime string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if any
	if entryID, exists := s.liveEntries[sourceID]; exists {
		s.cron.Remove(entryID)
		delete(s.liveEntries, sourceID)
	}

	cronExpr, err := BuildCronExpr(cronTime)
	if err != nil {
		return err
	}

	id := sourceID // Capture for closure
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		slog.Info("Cron: fetching live source", "id", id)
		if err := s.liveService.FetchAndUpdate(id); err != nil {
			slog.Error("Cron: failed to fetch live source", "id", id, "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.liveEntries[sourceID] = entryID
	slog.Info("Scheduled live source task", "id", sourceID, "interval", cronTime, "cron", cronExpr)
	return nil
}

// RemoveLiveSourceTask removes a cron job for a live source
func (s *Scheduler) RemoveLiveSourceTask(sourceID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.liveEntries[sourceID]; exists {
		s.cron.Remove(entryID)
		delete(s.liveEntries, sourceID)
		slog.Info("Removed live source task", "id", sourceID)
	}
}

// AddEPGSourceTask adds or updates a cron job for an EPG source
func (s *Scheduler) AddEPGSourceTask(sourceID uint, cronTime string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if any
	if entryID, exists := s.epgEntries[sourceID]; exists {
		s.cron.Remove(entryID)
		delete(s.epgEntries, sourceID)
	}

	cronExpr, err := BuildCronExpr(cronTime)
	if err != nil {
		return err
	}

	id := sourceID // Capture for closure
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		slog.Info("Cron: fetching EPG source", "id", id)
		if err := s.epgService.FetchAndUpdate(id); err != nil {
			slog.Error("Cron: failed to fetch EPG source", "id", id, "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.epgEntries[sourceID] = entryID
	slog.Info("Scheduled EPG source task", "id", sourceID, "interval", cronTime, "cron", cronExpr)
	return nil
}

// RemoveEPGSourceTask removes a cron job for an EPG source
func (s *Scheduler) RemoveEPGSourceTask(sourceID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.epgEntries[sourceID]; exists {
		s.cron.Remove(entryID)
		delete(s.epgEntries, sourceID)
		slog.Info("Removed EPG source task", "id", sourceID)
	}
}

// TriggerLiveSourceNow manually triggers a live source fetch immediately (for first-time add / manual refresh)
func (s *Scheduler) TriggerLiveSourceNow(sourceID uint) {
	go func() {
		slog.Info("Manual trigger: fetching live source", "id", sourceID)
		if err := s.liveService.FetchAndUpdate(sourceID); err != nil {
			slog.Error("Manual trigger: failed to fetch live source", "id", sourceID, "error", err)
		}
	}()
}

// TriggerEPGSourceNow manually triggers an EPG source fetch immediately
func (s *Scheduler) TriggerEPGSourceNow(sourceID uint) {
	go func() {
		slog.Info("Manual trigger: fetching EPG source", "id", sourceID)
		if err := s.epgService.FetchAndUpdate(sourceID); err != nil {
			slog.Error("Manual trigger: failed to fetch EPG source", "id", sourceID, "error", err)
		}
	}()
}

// AddDetectTask adds or updates a cron job for channel detection on a live source
func (s *Scheduler) AddDetectTask(sourceID uint, cronTime string, strategy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if any
	if entryID, exists := s.detectEntries[sourceID]; exists {
		s.cron.Remove(entryID)
		delete(s.detectEntries, sourceID)
	}

	cronExpr, err := BuildCronExpr(cronTime)
	if err != nil {
		return err
	}

	id := sourceID // Capture for closure
	st := strategy // Capture for closure
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		slog.Info("Cron: detecting channels for live source", "id", id, "strategy", st)
		if err := s.detectService.DetectChannels(id, false, st); err != nil {
			slog.Error("Cron: failed to detect channels", "id", id, "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add detect cron job: %w", err)
	}

	s.detectEntries[sourceID] = entryID
	slog.Info("Scheduled detect task", "id", sourceID, "interval", cronTime, "strategy", strategy, "cron", cronExpr)
	return nil
}

// RemoveDetectTask removes a cron job for channel detection
func (s *Scheduler) RemoveDetectTask(sourceID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.detectEntries[sourceID]; exists {
		s.cron.Remove(entryID)
		delete(s.detectEntries, sourceID)
		slog.Info("Removed detect task", "id", sourceID)
	}
}

// CheckFFprobe checks whether the ffprobe executable is available
func (s *Scheduler) CheckFFprobe() error {
	_, _, err := s.detectService.GetFFprobePath()
	return err
}

// TriggerDetectNow manually triggers channel detection immediately
func (s *Scheduler) TriggerDetectNow(sourceID uint, strategy string) {
	go func() {
		slog.Info("Manual trigger: detecting channels for live source", "id", sourceID, "strategy", strategy)
		if err := s.detectService.DetectChannels(sourceID, true, strategy); err != nil {
			slog.Error("Manual trigger: failed to detect channels", "id", sourceID, "error", err)
		}
	}()
}

// ValidateCronTime checks if a cronTime value is valid
func ValidateCronTime(cronTime string) bool {
	return validCronTimes[cronTime]
}
