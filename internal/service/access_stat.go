package service

import (
	"log/slog"
	"net/netip"
	"sync"
	"time"

	"iptv-tool-v2/internal/model"
)

// accessEvent represents a single IP access to be batched
type accessEvent struct {
	IP    string
	IsSub bool
	Time  time.Time
}

// AccessStatService manages IP access statistics with async batched writes
type AccessStatService struct {
	geoipSvc *GeoIPService
	eventCh  chan accessEvent
	stopCh   chan struct{}
	doneCh   chan struct{}
	wg       sync.WaitGroup
}

// NewAccessStatService creates and starts the background worker
func NewAccessStatService(geoipSvc *GeoIPService) *AccessStatService {
	svc := &AccessStatService{
		geoipSvc: geoipSvc,
		eventCh:  make(chan accessEvent, 1000),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
	svc.wg.Add(1)
	go svc.worker()
	return svc
}

// Record sends an access event to the background worker (non-blocking)
func (s *AccessStatService) Record(ip string, isSub bool) {
	select {
	case s.eventCh <- accessEvent{IP: ip, IsSub: isSub, Time: time.Now()}:
	default:
		// Channel full, drop event silently to avoid blocking the request
	}
}

// Stop gracefully stops the background worker
func (s *AccessStatService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// worker batches access events and flushes to DB periodically
func (s *AccessStatService) worker() {
	defer s.wg.Done()
	defer close(s.doneCh)

	// Aggregated events: ip -> {totalRequests, subRequests, lastTime}
	type aggEntry struct {
		totalRequests int64
		subRequests   int64
		lastTime      time.Time
	}
	batch := make(map[string]*aggEntry)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		for ip, agg := range batch {
			// Use raw SQL for UPSERT (SQLite ON CONFLICT)
			err := model.DB.Exec(`
				INSERT INTO access_stats (ip, total_requests, sub_requests, last_accessed_at)
				VALUES (?, ?, ?, ?)
				ON CONFLICT(ip) DO UPDATE SET
					total_requests = total_requests + excluded.total_requests,
					sub_requests = sub_requests + excluded.sub_requests,
					last_accessed_at = MAX(last_accessed_at, excluded.last_accessed_at)
			`, ip, agg.totalRequests, agg.subRequests, agg.lastTime).Error
			if err != nil {
				slog.Error("Failed to upsert access stat", "ip", ip, "error", err)
			}
		}

		// Clear batch
		batch = make(map[string]*aggEntry)
	}

	for {
		select {
		case <-s.stopCh:
			// Drain remaining events
			for {
				select {
				case evt := <-s.eventCh:
					entry, ok := batch[evt.IP]
					if !ok {
						entry = &aggEntry{}
						batch[evt.IP] = entry
					}
					entry.totalRequests++
					if evt.IsSub {
						entry.subRequests++
					}
					if evt.Time.After(entry.lastTime) {
						entry.lastTime = evt.Time
					}
				default:
					flush()
					return
				}
			}

		case evt := <-s.eventCh:
			entry, ok := batch[evt.IP]
			if !ok {
				entry = &aggEntry{}
				batch[evt.IP] = entry
			}
			entry.totalRequests++
			if evt.IsSub {
				entry.subRequests++
			}
			if evt.Time.After(entry.lastTime) {
				entry.lastTime = evt.Time
			}

			// Flush if batch is large
			if len(batch) >= 50 {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

// AccessStatItem represents a single row in the access stats response
type AccessStatItem struct {
	IP             string    `json:"ip"`
	Location       string    `json:"location"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
	TotalRequests  int64     `json:"total_requests"`
	SubRequests    int64     `json:"sub_requests"`
}

// Query returns paginated access stats from the last 7 days with GeoIP lookup
func (s *AccessStatService) Query(page, pageSize int, lang string, localLabel string) (items []AccessStatItem, total int64) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	cutoff := time.Now().AddDate(0, 0, -7)

	// Count total
	model.DB.Model(&model.AccessStat{}).
		Where("last_accessed_at >= ?", cutoff).
		Count(&total)

	// Fetch page
	var stats []model.AccessStat
	model.DB.Where("last_accessed_at >= ?", cutoff).
		Order("last_accessed_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&stats)

	items = make([]AccessStatItem, len(stats))
	for i, stat := range stats {
		location := ""
		if isPrivateOrLoopback(stat.IP) {
			location = localLabel
		} else if s.geoipSvc != nil && s.geoipSvc.DBExists() {
			country, province, city := s.geoipSvc.LookupIP(stat.IP, lang)
			location = buildLocation(country, province, city)
		}
		items[i] = AccessStatItem{
			IP:             stat.IP,
			Location:       location,
			LastAccessedAt: stat.LastAccessedAt,
			TotalRequests:  stat.TotalRequests,
			SubRequests:    stat.SubRequests,
		}
	}
	return items, total
}

// isPrivateOrLoopback checks if an IP is a private/loopback/link-local address
func isPrivateOrLoopback(ipStr string) bool {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}
	return addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast()
}

// Cleanup removes access stats older than 7 days
func (s *AccessStatService) Cleanup() {
	cutoff := time.Now().AddDate(0, 0, -7)
	result := model.DB.Where("last_accessed_at < ?", cutoff).Delete(&model.AccessStat{})
	if result.RowsAffected > 0 {
		slog.Info("Cleaned up old access stats", "deleted", result.RowsAffected)
	}
}

// buildLocation joins non-empty location parts
func buildLocation(country, province, city string) string {
	if country == "" && province == "" && city == "" {
		return ""
	}

	parts := make([]string, 0, 3)
	if country != "" {
		parts = append(parts, country)
	}
	if province != "" && province != country {
		parts = append(parts, province)
	}
	if city != "" && city != province {
		parts = append(parts, city)
	}

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}
