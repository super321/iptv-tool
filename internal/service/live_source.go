package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"iptv-tool-v2/internal/iptv"
	"iptv-tool-v2/internal/iptv/huawei"
	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/pkg/m3u"
)

// LiveSourceService handles fetching and updating live sources
type LiveSourceService struct{}

func NewLiveSourceService() *LiveSourceService {
	return &LiveSourceService{}
}

// FetchAndUpdate fetches the live source data and updates the database
func (s *LiveSourceService) FetchAndUpdate(sourceID uint) error {
	var source model.LiveSource
	if err := model.DB.First(&source, sourceID).Error; err != nil {
		return fmt.Errorf("live source %d not found: %w", sourceID, err)
	}

	if !source.Status {
		return nil // Source is disabled, skip
	}

	var channels []m3u.Channel
	var fetchErr error

	switch source.Type {
	case model.LiveSourceTypeIPTV:
		channels, fetchErr = s.fetchIPTV(source)
	case model.LiveSourceTypeNetworkURL:
		channels, fetchErr = s.fetchNetworkURL(source.URL)
	case model.LiveSourceTypeNetworkManual:
		channels, fetchErr = s.parseManualContent(source.Content)
	default:
		fetchErr = fmt.Errorf("unsupported live source type: %s", source.Type)
	}

	now := time.Now()
	if fetchErr != nil {
		// Update last error in database
		model.DB.Model(&source).Updates(map[string]interface{}{
			"last_error":      fetchErr.Error(),
			"last_fetched_at": &now,
		})
		return fetchErr
	}

	// Save parsed channels to database
	if err := s.saveParsedChannels(source.ID, channels); err != nil {
		return err
	}

	// Update last fetch time and clear error
	model.DB.Model(&source).Updates(map[string]interface{}{
		"last_fetched_at": &now,
		"last_error":      "",
	})

	log.Printf("Live source '%s' (ID: %d) fetched successfully, %d channels found.", source.Name, source.ID, len(channels))
	return nil
}

// ValidateNetworkURL validates a URL by fetching it and checking if the content is valid M3U or TXT format.
// Returns the detected format ("m3u" or "txt") and any error.
func (s *LiveSourceService) ValidateNetworkURL(sourceURL string) (string, error) {
	content, err := s.fetchURLContent(sourceURL)
	if err != nil {
		return "", err
	}

	format := m3u.DetectFormat(content)
	// Try to actually parse it to ensure it's valid
	var channels []m3u.Channel
	switch format {
	case "m3u":
		channels, err = m3u.ParseM3U(content)
	case "txt":
		channels, err = m3u.ParseTXT(content)
	default:
		return "", fmt.Errorf("content does not match M3U or TXT (DIYP) format")
	}
	if err != nil {
		return "", fmt.Errorf("content parsing failed: %w", err)
	}
	if len(channels) == 0 {
		return "", fmt.Errorf("no channels found in the content, please verify the URL")
	}

	return format, nil
}

// ValidateManualContent validates manually inputted content.
// Returns the detected format ("m3u" or "txt") and any error.
func (s *LiveSourceService) ValidateManualContent(content string) (string, error) {
	format := m3u.DetectFormat(content)
	var channels []m3u.Channel
	var err error
	switch format {
	case "m3u":
		channels, err = m3u.ParseM3U(content)
	case "txt":
		channels, err = m3u.ParseTXT(content)
	default:
		return "", fmt.Errorf("content does not match M3U or TXT (DIYP) format")
	}
	if err != nil {
		return "", fmt.Errorf("content parsing failed: %w", err)
	}
	if len(channels) == 0 {
		return "", fmt.Errorf("no channels found in the content")
	}
	return format, nil
}

func (s *LiveSourceService) fetchIPTV(source model.LiveSource) ([]m3u.Channel, error) {
	var config iptv.Config
	if err := json.Unmarshal([]byte(source.IPTVConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse IPTV config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Use factory pattern to create the right client based on platform
	client, err := createIPTVClient(&config)
	if err != nil {
		return nil, err
	}

	// Authenticate
	if err := client.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("IPTV authentication failed: %w", err)
	}

	// Fetch channel list
	iptvChannels, err := client.GetAllChannelList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IPTV channel list: %w", err)
	}

	// Convert iptv.Channel to m3u.Channel for unified storage
	var channels []m3u.Channel
	for _, ch := range iptvChannels {
		channels = append(channels, m3u.Channel{
			Name:        ch.Name,
			URL:         ch.URL,
			OriginalURL: ch.OriginalURL, // Make sure iptv implementation supplies this!
			TVGId:       ch.ID,
			TVGName:     ch.Name,
			CatchupSrc:  ch.CatchupURL,
			CatchupDays: ch.CatchupDays,
		})
	}

	return channels, nil
}

// createIPTVClient is a factory function that creates the correct IPTV client based on platform.
// Currently only supports Huawei; designed for extension to ZTE and others.
func createIPTVClient(config *iptv.Config) (iptv.Client, error) {
	switch strings.ToLower(config.Platform) {
	case "huawei", "hw", "hwctc":
		return huawei.NewClient(config), nil
	// case "zte":
	//     return zte.NewClient(config), nil
	default:
		return nil, fmt.Errorf("unsupported IPTV platform: %s", config.Platform)
	}
}

func (s *LiveSourceService) fetchNetworkURL(sourceURL string) ([]m3u.Channel, error) {
	content, err := s.fetchURLContent(sourceURL)
	if err != nil {
		return nil, err
	}
	return s.parseManualContent(content)
}

// fetchURLContent fetches URL content with a proper timeout to prevent goroutine leaks
func (s *LiveSourceService) fetchURLContent(sourceURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %w", sourceURL, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch URL returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func (s *LiveSourceService) parseManualContent(content string) ([]m3u.Channel, error) {
	format := m3u.DetectFormat(content)
	switch format {
	case "m3u":
		return m3u.ParseM3U(content)
	case "txt":
		return m3u.ParseTXT(content)
	default:
		return nil, fmt.Errorf("unrecognized content format")
	}
}

func (s *LiveSourceService) saveParsedChannels(sourceID uint, channels []m3u.Channel) error {
	// Delete old channels for this source
	if err := model.DB.Where("source_id = ?", sourceID).Delete(&model.ParsedChannel{}).Error; err != nil {
		return fmt.Errorf("failed to clear old channels: %w", err)
	}

	// Batch insert new channels
	var records []model.ParsedChannel
	for _, ch := range channels {
		records = append(records, model.ParsedChannel{
			SourceID:   sourceID,
			TVGId:      ch.TVGId,
			TVGName:    ch.TVGName,
			Name:       ch.Name,
			Group:      ch.Group,
			Logo:       ch.Logo,
			URL:        ch.URL,
			CatchupURL: ch.CatchupSrc,
		})
	}

	if len(records) > 0 {
		if err := model.DB.CreateInBatches(records, 100).Error; err != nil {
			return fmt.Errorf("failed to save parsed channels: %w", err)
		}
	}

	return nil
}
