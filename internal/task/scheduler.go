package task

import (
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/service"
)

// CronTimeMap maps the UI dropdown values to actual cron expressions.
// Minimum interval is 1 hour to avoid excessive requests to upstream servers.
var CronTimeMap = map[string]string{
	"1h":  "0 * * * *",    // Every hour at minute 0
	"2h":  "0 */2 * * *",  // Every 2 hours
	"4h":  "0 */4 * * *",  // Every 4 hours
	"6h":  "0 */6 * * *",  // Every 6 hours
	"12h": "0 */12 * * *", // Every 12 hours
	"24h": "0 0 * * *",    // Every day at midnight
}

// CronTimeOptions returns the available options for the frontend dropdown
var CronTimeOptions = []map[string]string{
	{"value": "1h", "label": "每1小时"},
	{"value": "2h", "label": "每2小时"},
	{"value": "4h", "label": "每4小时"},
	{"value": "6h", "label": "每6小时"},
	{"value": "12h", "label": "每12小时"},
	{"value": "24h", "label": "每天"},
}

// Scheduler manages all cron jobs for live sources and EPG sources
type Scheduler struct {
	cron        *cron.Cron
	liveService *service.LiveSourceService
	epgService  *service.EPGSourceService

	mu          sync.Mutex
	liveEntries map[uint]cron.EntryID // sourceID -> cron entry ID
	epgEntries  map[uint]cron.EntryID // sourceID -> cron entry ID
}

// NewScheduler creates a new task scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron:        cron.New(),
		liveService: service.NewLiveSourceService(),
		epgService:  service.NewEPGSourceService(),
		liveEntries: make(map[uint]cron.EntryID),
		epgEntries:  make(map[uint]cron.EntryID),
	}
}

// Start initializes and starts all scheduled tasks from the database
func (s *Scheduler) Start() error {
	log.Println("Initializing task scheduler...")

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
			log.Printf("Warning: failed to schedule live source '%s' (ID: %d): %v", src.Name, src.ID, err)
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
			log.Printf("Warning: failed to schedule EPG source '%s' (ID: %d): %v", src.Name, src.ID, err)
		}
	}

	s.cron.Start()
	log.Printf("Task scheduler started: %d live source tasks, %d EPG source tasks.",
		len(s.liveEntries), len(s.epgEntries))
	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Task scheduler stopped.")
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

	cronExpr, ok := CronTimeMap[cronTime]
	if !ok {
		return fmt.Errorf("invalid cron time: %s", cronTime)
	}

	id := sourceID // Capture for closure
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		log.Printf("Cron: fetching live source ID: %d", id)
		if err := s.liveService.FetchAndUpdate(id); err != nil {
			log.Printf("Cron: failed to fetch live source ID %d: %v", id, err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.liveEntries[sourceID] = entryID
	log.Printf("Scheduled live source task: ID=%d, interval=%s, cron=%s", sourceID, cronTime, cronExpr)
	return nil
}

// RemoveLiveSourceTask removes a cron job for a live source
func (s *Scheduler) RemoveLiveSourceTask(sourceID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.liveEntries[sourceID]; exists {
		s.cron.Remove(entryID)
		delete(s.liveEntries, sourceID)
		log.Printf("Removed live source task: ID=%d", sourceID)
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

	cronExpr, ok := CronTimeMap[cronTime]
	if !ok {
		return fmt.Errorf("invalid cron time: %s", cronTime)
	}

	id := sourceID // Capture for closure
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		log.Printf("Cron: fetching EPG source ID: %d", id)
		if err := s.epgService.FetchAndUpdate(id); err != nil {
			log.Printf("Cron: failed to fetch EPG source ID %d: %v", id, err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.epgEntries[sourceID] = entryID
	log.Printf("Scheduled EPG source task: ID=%d, interval=%s, cron=%s", sourceID, cronTime, cronExpr)
	return nil
}

// RemoveEPGSourceTask removes a cron job for an EPG source
func (s *Scheduler) RemoveEPGSourceTask(sourceID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.epgEntries[sourceID]; exists {
		s.cron.Remove(entryID)
		delete(s.epgEntries, sourceID)
		log.Printf("Removed EPG source task: ID=%d", sourceID)
	}
}

// TriggerLiveSourceNow manually triggers a live source fetch immediately (for first-time add / manual refresh)
func (s *Scheduler) TriggerLiveSourceNow(sourceID uint) {
	go func() {
		log.Printf("Manual trigger: fetching live source ID: %d", sourceID)
		if err := s.liveService.FetchAndUpdate(sourceID); err != nil {
			log.Printf("Manual trigger: failed to fetch live source ID %d: %v", sourceID, err)
		}
	}()
}

// TriggerEPGSourceNow manually triggers an EPG source fetch immediately
func (s *Scheduler) TriggerEPGSourceNow(sourceID uint) {
	go func() {
		log.Printf("Manual trigger: fetching EPG source ID: %d", sourceID)
		if err := s.epgService.FetchAndUpdate(sourceID); err != nil {
			log.Printf("Manual trigger: failed to fetch EPG source ID %d: %v", sourceID, err)
		}
	}()
}

// ValidateCronTime checks if a cronTime value is valid
func ValidateCronTime(cronTime string) bool {
	_, ok := CronTimeMap[cronTime]
	return ok
}
