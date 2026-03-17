package task

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/publish"
	"iptv-tool-v2/internal/service"
)

// validIntervals is the set of valid user-facing interval keys.
var validIntervals = map[string]bool{
	"1h": true, "2h": true, "4h": true,
	"6h": true, "12h": true, "24h": true,
}

// ParseInterval converts an interval key (e.g. "1h", "2h", "24h") to a time.Duration.
func ParseInterval(interval string) (time.Duration, error) {
	if !validIntervals[interval] {
		return 0, fmt.Errorf("invalid interval: %s", interval)
	}
	return time.ParseDuration(interval)
}

// IntervalOptions returns the available options for the frontend dropdown.
// Labels are i18n keys that should be translated at the handler level.
var IntervalOptions = []map[string]string{
	{"value": "1h", "label": "label.interval_1h"},
	{"value": "2h", "label": "label.interval_2h"},
	{"value": "4h", "label": "label.interval_4h"},
	{"value": "6h", "label": "label.interval_6h"},
	{"value": "12h", "label": "label.interval_12h"},
	{"value": "24h", "label": "label.interval_24h"},
}

// taskHandle holds a stop channel and a done channel for a scheduled task goroutine.
// stopCh is closed to signal the goroutine to stop.
// doneCh is closed by the goroutine when it has fully exited.
type taskHandle struct {
	stopCh chan struct{}
	doneCh chan struct{}
}

// Scheduler manages all interval-based scheduled tasks for live sources and EPG sources
type Scheduler struct {
	liveService   *service.LiveSourceService
	epgService    *service.EPGSourceService
	detectService *service.DetectService

	mu            sync.Mutex
	wg            sync.WaitGroup
	liveEntries   map[uint]*taskHandle // sourceID -> task handle
	epgEntries    map[uint]*taskHandle // sourceID -> task handle
	detectEntries map[uint]*taskHandle // sourceID -> detect task handle
}

// NewScheduler creates a new task scheduler
func NewScheduler(dataDir string) *Scheduler {
	return &Scheduler{
		liveService:   service.NewLiveSourceService(),
		epgService:    service.NewEPGSourceService(),
		detectService: service.NewDetectService(dataDir),
		liveEntries:   make(map[uint]*taskHandle),
		epgEntries:    make(map[uint]*taskHandle),
		detectEntries: make(map[uint]*taskHandle),
	}
}

// stopAndWait signals a task to stop and waits for its goroutine to fully exit.
// The caller must NOT hold s.mu when calling this function.
func stopAndWait(h *taskHandle) {
	close(h.stopCh)
	<-h.doneCh // Wait for goroutine to fully exit
}

// Start initializes and starts all scheduled tasks from the database
func (s *Scheduler) Start() error {
	slog.Info("Initializing task scheduler...")

	// Load all enabled live sources and register their scheduled tasks
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

	// Load all enabled EPG sources and register their scheduled tasks
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

	// Load all enabled live sources with detect interval and register their detect tasks
	var detectSources []model.LiveSource
	if err := model.DB.Where("status = ? AND cron_detect != ''", true).Find(&detectSources).Error; err != nil {
		return fmt.Errorf("failed to load live sources for detection: %w", err)
	}

	for _, src := range detectSources {
		if err := s.AddDetectTask(src.ID, src.CronDetect, src.DetectStrategy); err != nil {
			slog.Warn("Failed to schedule detect task", "name", src.Name, "id", src.ID, "error", err)
		}
	}

	s.mu.Lock()
	liveCount := len(s.liveEntries)
	epgCount := len(s.epgEntries)
	detectCount := len(s.detectEntries)
	s.mu.Unlock()
	slog.Info("Task scheduler started", "live_tasks", liveCount, "epg_tasks", epgCount, "detect_tasks", detectCount)
	return nil
}

// Stop gracefully stops the scheduler, waiting for all running task goroutines to finish
func (s *Scheduler) Stop() {
	s.mu.Lock()
	// Collect all handles first, then close them outside the lock to avoid
	// holding the lock while waiting for goroutines
	var handles []*taskHandle
	for id, h := range s.liveEntries {
		handles = append(handles, h)
		delete(s.liveEntries, id)
	}
	for id, h := range s.epgEntries {
		handles = append(handles, h)
		delete(s.epgEntries, id)
	}
	for id, h := range s.detectEntries {
		handles = append(handles, h)
		delete(s.detectEntries, id)
	}
	s.mu.Unlock()

	// Signal all goroutines to stop
	for _, h := range handles {
		close(h.stopCh)
	}

	s.wg.Wait()
	slog.Info("Task scheduler stopped.")
}

// AddLiveSourceTask adds or updates a scheduled task for a live source
func (s *Scheduler) AddLiveSourceTask(sourceID uint, interval string) error {
	dur, err := ParseInterval(interval)
	if err != nil {
		return err
	}

	// Extract old handle under lock, then wait for it outside the lock
	s.mu.Lock()
	oldHandle := s.liveEntries[sourceID]
	delete(s.liveEntries, sourceID)
	s.mu.Unlock()

	// Wait for old goroutine to fully exit (outside lock to avoid deadlock)
	if oldHandle != nil {
		stopAndWait(oldHandle)
	}

	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	handle := &taskHandle{stopCh: stopCh, doneCh: doneCh}

	s.mu.Lock()
	s.liveEntries[sourceID] = handle
	s.mu.Unlock()

	id := sourceID // Capture for closure
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(doneCh)
		ticker := time.NewTicker(dur)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				slog.Info("Scheduler: fetching live source", "id", id)
				if err := s.liveService.FetchAndUpdate(id); err != nil {
					slog.Error("Scheduler: failed to fetch live source", "id", id, "error", err)
				} else {
					publish.InvalidateAll()
				}
			}
		}
	}()

	slog.Info("Scheduled live source task", "id", sourceID, "interval", interval)
	return nil
}

// RemoveLiveSourceTask removes a scheduled task for a live source
func (s *Scheduler) RemoveLiveSourceTask(sourceID uint) {
	s.mu.Lock()
	h := s.liveEntries[sourceID]
	delete(s.liveEntries, sourceID)
	s.mu.Unlock()

	if h != nil {
		stopAndWait(h)
		slog.Info("Removed live source task", "id", sourceID)
	}
}

// AddEPGSourceTask adds or updates a scheduled task for an EPG source
func (s *Scheduler) AddEPGSourceTask(sourceID uint, interval string) error {
	dur, err := ParseInterval(interval)
	if err != nil {
		return err
	}

	// Extract old handle under lock, then wait for it outside the lock
	s.mu.Lock()
	oldHandle := s.epgEntries[sourceID]
	delete(s.epgEntries, sourceID)
	s.mu.Unlock()

	// Wait for old goroutine to fully exit (outside lock to avoid deadlock)
	if oldHandle != nil {
		stopAndWait(oldHandle)
	}

	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	handle := &taskHandle{stopCh: stopCh, doneCh: doneCh}

	s.mu.Lock()
	s.epgEntries[sourceID] = handle
	s.mu.Unlock()

	id := sourceID // Capture for closure
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(doneCh)
		ticker := time.NewTicker(dur)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				slog.Info("Scheduler: fetching EPG source", "id", id)
				if err := s.epgService.FetchAndUpdate(id); err != nil {
					slog.Error("Scheduler: failed to fetch EPG source", "id", id, "error", err)
				} else {
					publish.InvalidateAll()
				}
			}
		}
	}()

	slog.Info("Scheduled EPG source task", "id", sourceID, "interval", interval)
	return nil
}

// RemoveEPGSourceTask removes a scheduled task for an EPG source
func (s *Scheduler) RemoveEPGSourceTask(sourceID uint) {
	s.mu.Lock()
	h := s.epgEntries[sourceID]
	delete(s.epgEntries, sourceID)
	s.mu.Unlock()

	if h != nil {
		stopAndWait(h)
		slog.Info("Removed EPG source task", "id", sourceID)
	}
}

// TriggerLiveSourceNow manually triggers a live source fetch immediately (for first-time add / manual refresh)
func (s *Scheduler) TriggerLiveSourceNow(sourceID uint) {
	go func() {
		slog.Info("Manual trigger: fetching live source", "id", sourceID)
		if err := s.liveService.FetchAndUpdate(sourceID); err != nil {
			slog.Error("Manual trigger: failed to fetch live source", "id", sourceID, "error", err)
		} else {
			publish.InvalidateAll()
		}
	}()
}

// TriggerEPGSourceNow manually triggers an EPG source fetch immediately
func (s *Scheduler) TriggerEPGSourceNow(sourceID uint) {
	go func() {
		slog.Info("Manual trigger: fetching EPG source", "id", sourceID)
		if err := s.epgService.FetchAndUpdate(sourceID); err != nil {
			slog.Error("Manual trigger: failed to fetch EPG source", "id", sourceID, "error", err)
		} else {
			publish.InvalidateAll()
		}
	}()
}

// AddDetectTask adds or updates a scheduled task for channel detection on a live source
func (s *Scheduler) AddDetectTask(sourceID uint, interval string, strategy string) error {
	dur, err := ParseInterval(interval)
	if err != nil {
		return err
	}

	// Extract old handle under lock, then wait for it outside the lock
	s.mu.Lock()
	oldHandle := s.detectEntries[sourceID]
	delete(s.detectEntries, sourceID)
	s.mu.Unlock()

	// Wait for old goroutine to fully exit (outside lock to avoid deadlock)
	if oldHandle != nil {
		stopAndWait(oldHandle)
	}

	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	handle := &taskHandle{stopCh: stopCh, doneCh: doneCh}

	s.mu.Lock()
	s.detectEntries[sourceID] = handle
	s.mu.Unlock()

	id := sourceID // Capture for closure
	st := strategy // Capture for closure
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(doneCh)
		ticker := time.NewTicker(dur)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				slog.Info("Scheduler: detecting channels for live source", "id", id, "strategy", st)
				if err := s.detectService.DetectChannels(id, false, st); err != nil {
					slog.Error("Scheduler: failed to detect channels", "id", id, "error", err)
				} else {
					publish.InvalidateAll()
				}
			}
		}
	}()

	slog.Info("Scheduled detect task", "id", sourceID, "interval", interval, "strategy", strategy)
	return nil
}

// RemoveDetectTask removes a scheduled task for channel detection
func (s *Scheduler) RemoveDetectTask(sourceID uint) {
	s.mu.Lock()
	h := s.detectEntries[sourceID]
	delete(s.detectEntries, sourceID)
	s.mu.Unlock()

	if h != nil {
		stopAndWait(h)
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
		} else {
			publish.InvalidateAll()
		}
	}()
}

// ValidateInterval checks if an interval value is valid
func ValidateInterval(interval string) bool {
	return validIntervals[interval]
}
