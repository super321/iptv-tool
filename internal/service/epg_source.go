package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"iptv-tool-v2/internal/iptv"
	"iptv-tool-v2/internal/model"
	epgpkg "iptv-tool-v2/pkg/epg"
)

// EPGSourceService handles fetching and updating EPG sources
type EPGSourceService struct{}

func NewEPGSourceService() *EPGSourceService {
	return &EPGSourceService{}
}

// FetchAndUpdate fetches the EPG source data and updates the database
func (s *EPGSourceService) FetchAndUpdate(sourceID uint) error {
	var source model.EPGSource
	if err := model.DB.First(&source, sourceID).Error; err != nil {
		return fmt.Errorf("epg source %d not found: %w", sourceID, err)
	}

	if !source.Status {
		return nil // Source is disabled, skip
	}

	// Mark as syncing in case this was triggered by a cron job
	model.DB.Model(&source).Update("is_syncing", true)

	defer func() {
		// Defensive cleanup
		model.DB.Model(&model.EPGSource{}).Where("id = ?", sourceID).Update("is_syncing", false)
	}()

	var programs []epgpkg.Program
	var fetchErr error

	switch source.Type {
	case model.EPGSourceTypeNetworkXMLTV:
		programs, fetchErr = s.fetchNetworkXMLTV(source.URL)
	case model.EPGSourceTypeIPTV:
		// Wait for associated live source to finish syncing to avoid auth token conflicts
		if source.LiveSourceID != nil {
			if err := s.WaitForLiveSourceSyncComplete(*source.LiveSourceID, 10*time.Minute); err != nil {
				fetchErr = err
			}
		}
		if fetchErr == nil {
			programs, fetchErr = s.fetchIPTVEPG(source)
		}
	default:
		fetchErr = fmt.Errorf("unsupported EPG source type: %s", source.Type)
	}

	now := time.Now()
	if fetchErr != nil {
		model.DB.Model(&source).Updates(map[string]interface{}{
			"last_error":      fetchErr.Error(),
			"last_fetched_at": &now,
		})
		return fetchErr
	}

	// Save parsed EPG to database
	if err := s.saveParsedEPG(source.ID, programs); err != nil {
		return err
	}

	// Update last fetch time and clear error
	model.DB.Model(&source).Updates(map[string]interface{}{
		"last_fetched_at": &now,
		"last_error":      "",
	})

	slog.Info("EPG source fetched successfully", "name", source.Name, "id", source.ID, "programs", len(programs))
	return nil
}

func (s *EPGSourceService) fetchNetworkXMLTV(url string) ([]epgpkg.Program, error) {
	// FetchAndParseXMLTV automatically handles gzip detection (via HTTP Header and magic number)
	programs, err := epgpkg.FetchAndParseXMLTV(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch/parse XMLTV from %s: %w", url, err)
	}
	return programs, nil
}

func (s *EPGSourceService) fetchIPTVEPG(source model.EPGSource) ([]epgpkg.Program, error) {
	var config iptv.Config
	if err := json.Unmarshal([]byte(source.IPTVConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse IPTV EPG config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create IPTV client using the factory (pass pointer so strategy auto-detect can write back)
	client, err := createIPTVClient(&config)
	if err != nil {
		return nil, err
	}

	// Authenticate first
	if err := client.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("IPTV authentication failed: %w", err)
	}

	// Get channel list (needed to know which channels to fetch EPG for)
	iptvChannels, err := client.GetAllChannelList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channel list for EPG: %w", err)
	}

	// Fetch EPG using the configured strategy (with rate limiting and retry built in)
	progLists, err := client.GetAllChannelProgramList(ctx, iptvChannels)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IPTV EPG: %w", err)
	}

	// Convert iptv.ChannelProgramList to epgpkg.Program for unified storage
	var programs []epgpkg.Program
	for _, pl := range progLists {
		for _, prog := range pl.Programs {
			programs = append(programs, epgpkg.Program{
				Channel:     pl.Channel.ID,
				ChannelName: pl.Channel.Name,
				Title:       prog.Title,
				Desc:        prog.Desc,
				StartTime:   prog.StartTime,
				EndTime:     prog.EndTime,
			})
		}
	}

	// If auto-detect found a working strategy, persist it back so next run skips detection
	// config is a local variable, but the Client holds a pointer to it,
	// so changes from autoDetectEPGStrategy are reflected here.
	if config.EPGStrategy != "" && config.EPGStrategy != "auto" {
		configJSON, _ := json.Marshal(config)
		model.DB.Model(&source).Update("iptv_config", string(configJSON))
	}

	return programs, nil
}

func (s *EPGSourceService) saveParsedEPG(sourceID uint, programs []epgpkg.Program) error {
	// Delete old EPG for this source
	if err := model.DB.Where("source_id = ?", sourceID).Delete(&model.ParsedEPG{}).Error; err != nil {
		return fmt.Errorf("failed to clear old EPG data: %w", err)
	}

	// Batch insert new EPG records
	var records []model.ParsedEPG
	for _, prog := range programs {
		records = append(records, model.ParsedEPG{
			SourceID:    sourceID,
			Channel:     strings.TrimSpace(prog.Channel),
			ChannelName: strings.TrimSpace(prog.ChannelName),
			Title:       prog.Title,
			Desc:        prog.Desc,
			StartTime:   prog.StartTime,
			EndTime:     prog.EndTime,
		})
	}

	if len(records) > 0 {
		if err := model.DB.CreateInBatches(records, 200).Error; err != nil {
			return fmt.Errorf("failed to save parsed EPG: %w", err)
		}
	}

	return nil
}

// WaitForLiveSourceSyncComplete polls the associated live source's is_syncing status
// until it becomes false or maxWait duration is reached.
func (s *EPGSourceService) WaitForLiveSourceSyncComplete(sourceID uint, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		var lSource model.LiveSource
		if err := model.DB.First(&lSource, sourceID).Error; err != nil {
			// If not found, skip wait
			return nil
		}
		if !lSource.IsSyncing {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("等待关联的首发直播源刷新完成超时（超过 %v）", maxWait)
		}
		<-ticker.C
	}
}
