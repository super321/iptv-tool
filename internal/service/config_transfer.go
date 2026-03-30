package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/internal/version"
)

// --- Export/Import module constants ---

const (
	ModuleSources       = "sources"
	ModuleLogos         = "logos"
	ModuleRules         = "rules"
	ModulePublish       = "publish"
	ModuleDetect        = "detect"
	ModuleAccessControl = "access_control"
)

// AllModules lists all supported modules in dependency order.
var AllModules = []string{ModuleSources, ModuleLogos, ModuleRules, ModulePublish, ModuleDetect, ModuleAccessControl}

// --- Manifest ---

// ExportManifest is written as manifest.json inside the ZIP.
type ExportManifest struct {
	Version    string   `json:"version"`
	ExportedAt string   `json:"exported_at"`
	Modules    []string `json:"modules"`
}

// --- Import summary / result types ---

// ImportModuleSummary describes one module found inside a parsed ZIP.
type ImportModuleSummary struct {
	Module string `json:"module"`
	Count  int    `json:"count"`
}

// ExportLogo wraps ChannelLogo for export because FilePath has json:"-" tag.
// We store the base filename so the import can match ZIP entries.
type ExportLogo struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	FileName string `json:"file_name"` // Base filename on disk (e.g. "CCTV1.png")
	URLPath  string `json:"url_path"`
}

// ImportParsedData holds all parsed data ready for execution.
type ImportParsedData struct {
	Manifest       ExportManifest             `json:"manifest"`
	Summaries      []ImportModuleSummary      `json:"summaries"`
	LiveSources    []model.LiveSource         `json:"-"`
	EPGSources     []model.EPGSource          `json:"-"`
	Logos          []ExportLogo               `json:"-"`
	LogoFiles      map[string][]byte          `json:"-"` // fileName -> fileBytes
	Rules          []model.AggregationRule    `json:"-"`
	PublishIfaces  []model.PublishInterface   `json:"-"`
	DetectSettings map[string]string          `json:"-"` // key -> value
	ACLMode        string                     `json:"-"`
	ACLEntries     []model.AccessControlEntry `json:"-"`
}

// ModuleResult reports the import outcome for a single module.
type ModuleResult struct {
	Module  string   `json:"module"`
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Skipped int      `json:"skipped"`
	Details []string `json:"details"`
}

// ImportResult is the overall import response.
type ImportResult struct {
	Modules []ModuleResult `json:"modules"`
}

// TaskScheduler abstracts the scheduler operations needed by import/export.
// This avoids an import cycle between service and task packages.
type TaskScheduler interface {
	RemoveLiveSourceTask(sourceID uint)
	RemoveDetectTask(sourceID uint)
	RemoveEPGSourceTask(sourceID uint)
	AddLiveSourceTask(sourceID uint, cfg *model.ScheduleConfig) error
	AddDetectTask(sourceID uint, cfg *model.ScheduleConfig, detectStrategy string) error
	AddEPGSourceTask(sourceID uint, cfg *model.ScheduleConfig) error
}

// --- Service ---

// ConfigTransferService handles export and import of system configuration.
type ConfigTransferService struct {
	logoDir   string
	scheduler TaskScheduler
}

// NewConfigTransferService creates a new service instance.
func NewConfigTransferService(logoDir string, scheduler TaskScheduler) *ConfigTransferService {
	return &ConfigTransferService{
		logoDir:   logoDir,
		scheduler: scheduler,
	}
}

// ============================================================================
// EXPORT
// ============================================================================

// ExportConfig builds a ZIP archive containing the requested modules.
func (s *ConfigTransferService) ExportConfig(modules []string) (*bytes.Buffer, error) {
	moduleSet := make(map[string]bool)
	for _, m := range modules {
		moduleSet[m] = true
	}

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	manifest := ExportManifest{
		Version:    version.Version,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Modules:    modules,
	}

	// --- Sources ---
	if moduleSet[ModuleSources] {
		var liveSources []model.LiveSource
		model.DB.Find(&liveSources)
		if err := writeJSON(zw, "iptv-config/live_sources.json", liveSources); err != nil {
			return nil, fmt.Errorf("write live_sources.json: %w", err)
		}

		var epgSources []model.EPGSource
		model.DB.Find(&epgSources)
		if err := writeJSON(zw, "iptv-config/epg_sources.json", epgSources); err != nil {
			return nil, fmt.Errorf("write epg_sources.json: %w", err)
		}
	}

	// --- Logos ---
	if moduleSet[ModuleLogos] {
		var logos []model.ChannelLogo
		model.DB.Find(&logos)

		// Build export structs with filename (FilePath has json:"-")
		exportLogos := make([]ExportLogo, 0, len(logos))
		for _, logo := range logos {
			exportLogos = append(exportLogos, ExportLogo{
				ID:       logo.ID,
				Name:     logo.Name,
				FileName: filepath.Base(logo.FilePath),
				URLPath:  logo.URLPath,
			})
		}
		if err := writeJSON(zw, "iptv-config/logos/logo_list.json", exportLogos); err != nil {
			return nil, fmt.Errorf("write logo_list.json: %w", err)
		}

		// Copy logo files into ZIP
		for _, logo := range logos {
			fileData, err := os.ReadFile(logo.FilePath)
			if err != nil {
				slog.Warn("Export: skipping missing logo file", "path", logo.FilePath, "error", err)
				continue
			}
			fileName := filepath.Base(logo.FilePath)
			fw, err := zw.Create("iptv-config/logos/files/" + fileName)
			if err != nil {
				return nil, fmt.Errorf("create logo file in zip: %w", err)
			}
			if _, err := fw.Write(fileData); err != nil {
				return nil, fmt.Errorf("write logo file: %w", err)
			}
		}
	}

	// --- Rules ---
	if moduleSet[ModuleRules] {
		var rules []model.AggregationRule
		model.DB.Find(&rules)
		if err := writeJSON(zw, "iptv-config/rules.json", rules); err != nil {
			return nil, fmt.Errorf("write rules.json: %w", err)
		}
	}

	// --- Publish ---
	if moduleSet[ModulePublish] {
		var ifaces []model.PublishInterface
		model.DB.Find(&ifaces)
		if err := writeJSON(zw, "iptv-config/publish_interfaces.json", ifaces); err != nil {
			return nil, fmt.Errorf("write publish_interfaces.json: %w", err)
		}
	}

	// --- Detect Settings ---
	if moduleSet[ModuleDetect] {
		var settings []model.SystemSetting
		model.DB.Where("key IN ?", []string{"detect_concurrency", "detect_timeout"}).Find(&settings)
		m := make(map[string]string)
		for _, s := range settings {
			m[s.Key] = s.Value
		}
		if err := writeJSON(zw, "iptv-config/detect_settings.json", m); err != nil {
			return nil, fmt.Errorf("write detect_settings.json: %w", err)
		}
	}

	// --- Access Control ---
	if moduleSet[ModuleAccessControl] {
		mode := "disabled"
		var modeSetting model.SystemSetting
		if err := model.DB.Where("key = ?", "access_control_mode").First(&modeSetting).Error; err == nil {
			mode = modeSetting.Value
		}
		var entries []model.AccessControlEntry
		model.DB.Find(&entries)

		aclData := map[string]interface{}{
			"mode":    mode,
			"entries": entries,
		}
		if err := writeJSON(zw, "iptv-config/access_control.json", aclData); err != nil {
			return nil, fmt.Errorf("write access_control.json: %w", err)
		}
	}

	// --- Manifest ---
	if err := writeJSON(zw, "iptv-config/manifest.json", manifest); err != nil {
		return nil, fmt.Errorf("write manifest.json: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}

	return buf, nil
}

// ============================================================================
// IMPORT — Parse
// ============================================================================

// ParseImportZip reads and validates a ZIP, returning a summary for user confirmation.
func (s *ConfigTransferService) ParseImportZip(zipData []byte) (*ImportParsedData, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip: %w", err)
	}

	// Build file-lookup map (strip optional top-level folder)
	files := make(map[string]*zip.File)
	for _, f := range zr.File {
		name := f.Name
		// Normalise: strip leading "iptv-config/"
		name = strings.TrimPrefix(name, "iptv-config/")
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		files[name] = f
	}

	// Manifest
	manifestFile, ok := files["manifest.json"]
	if !ok {
		return nil, fmt.Errorf("missing manifest.json")
	}
	var manifest ExportManifest
	if err := readZipJSON(manifestFile, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest.json: %w", err)
	}

	data := &ImportParsedData{
		Manifest:  manifest,
		LogoFiles: make(map[string][]byte),
	}

	moduleSet := make(map[string]bool)
	for _, m := range manifest.Modules {
		moduleSet[m] = true
	}

	// Sources
	if moduleSet[ModuleSources] {
		if f, ok := files["live_sources.json"]; ok {
			if err := readZipJSON(f, &data.LiveSources); err != nil {
				return nil, fmt.Errorf("parse live_sources.json: %w", err)
			}
		}
		if f, ok := files["epg_sources.json"]; ok {
			if err := readZipJSON(f, &data.EPGSources); err != nil {
				return nil, fmt.Errorf("parse epg_sources.json: %w", err)
			}
		}
		data.Summaries = append(data.Summaries, ImportModuleSummary{
			Module: ModuleSources,
			Count:  len(data.LiveSources) + len(data.EPGSources),
		})
	}

	// Logos
	if moduleSet[ModuleLogos] {
		if f, ok := files["logos/logo_list.json"]; ok {
			if err := readZipJSON(f, &data.Logos); err != nil {
				return nil, fmt.Errorf("parse logo_list.json: %w", err)
			}
		}
		// Read logo files
		for name, f := range files {
			if strings.HasPrefix(name, "logos/files/") {
				raw, err := readZipRaw(f)
				if err != nil {
					slog.Warn("Import: skip unreadable logo file", "name", name, "error", err)
					continue
				}
				fileName := strings.TrimPrefix(name, "logos/files/")
				data.LogoFiles[fileName] = raw
			}
		}
		data.Summaries = append(data.Summaries, ImportModuleSummary{
			Module: ModuleLogos,
			Count:  len(data.Logos),
		})
	}

	// Rules
	if moduleSet[ModuleRules] {
		if f, ok := files["rules.json"]; ok {
			if err := readZipJSON(f, &data.Rules); err != nil {
				return nil, fmt.Errorf("parse rules.json: %w", err)
			}
		}
		data.Summaries = append(data.Summaries, ImportModuleSummary{
			Module: ModuleRules,
			Count:  len(data.Rules),
		})
	}

	// Publish
	if moduleSet[ModulePublish] {
		if f, ok := files["publish_interfaces.json"]; ok {
			if err := readZipJSON(f, &data.PublishIfaces); err != nil {
				return nil, fmt.Errorf("parse publish_interfaces.json: %w", err)
			}
		}
		data.Summaries = append(data.Summaries, ImportModuleSummary{
			Module: ModulePublish,
			Count:  len(data.PublishIfaces),
		})
	}

	// Detect
	if moduleSet[ModuleDetect] {
		if f, ok := files["detect_settings.json"]; ok {
			data.DetectSettings = make(map[string]string)
			if err := readZipJSON(f, &data.DetectSettings); err != nil {
				return nil, fmt.Errorf("parse detect_settings.json: %w", err)
			}
		}
		data.Summaries = append(data.Summaries, ImportModuleSummary{
			Module: ModuleDetect,
			Count:  len(data.DetectSettings),
		})
	}

	// Access Control
	if moduleSet[ModuleAccessControl] {
		if f, ok := files["access_control.json"]; ok {
			var aclRaw struct {
				Mode    string                     `json:"mode"`
				Entries []model.AccessControlEntry `json:"entries"`
			}
			if err := readZipJSON(f, &aclRaw); err != nil {
				return nil, fmt.Errorf("parse access_control.json: %w", err)
			}
			data.ACLMode = aclRaw.Mode
			data.ACLEntries = aclRaw.Entries
		}
		data.Summaries = append(data.Summaries, ImportModuleSummary{
			Module: ModuleAccessControl,
			Count:  len(data.ACLEntries) + 1, // +1 for the mode setting
		})
	}

	return data, nil
}

// ============================================================================
// IMPORT — Execute
// ============================================================================

// ExecuteImport performs the actual data import and returns per-module results.
func (s *ConfigTransferService) ExecuteImport(data *ImportParsedData) *ImportResult {
	result := &ImportResult{}

	moduleSet := make(map[string]bool)
	for _, m := range data.Manifest.Modules {
		moduleSet[m] = true
	}

	// ID mapping tables: oldID -> newID
	liveSourceIDMap := make(map[uint]uint)
	epgSourceIDMap := make(map[uint]uint)
	ruleIDMap := make(map[uint]uint)

	// Build a name->LiveSource map of existing live sources for EPG linking
	liveSourceNameMap := make(map[string]uint)

	// 1. Sources
	if moduleSet[ModuleSources] {
		mr := s.importSources(data, liveSourceIDMap, epgSourceIDMap, liveSourceNameMap)
		result.Modules = append(result.Modules, mr)
	}

	// 2. Logos
	if moduleSet[ModuleLogos] {
		mr := s.importLogos(data)
		result.Modules = append(result.Modules, mr)
	}

	// 3. Rules
	if moduleSet[ModuleRules] {
		mr := s.importRules(data, ruleIDMap)
		result.Modules = append(result.Modules, mr)
	}

	// 4. Publish
	if moduleSet[ModulePublish] {
		mr := s.importPublish(data, liveSourceIDMap, epgSourceIDMap, ruleIDMap)
		result.Modules = append(result.Modules, mr)
	}

	// 5. Detect
	if moduleSet[ModuleDetect] {
		mr := s.importDetect(data)
		result.Modules = append(result.Modules, mr)
	}

	// 6. Access Control
	if moduleSet[ModuleAccessControl] {
		mr := s.importAccessControl(data)
		result.Modules = append(result.Modules, mr)
	}

	return result
}

// --- Import helpers per module ---

func (s *ConfigTransferService) importSources(data *ImportParsedData, liveIDMap, epgIDMap map[uint]uint, liveNameMap map[string]uint) ModuleResult {
	mr := ModuleResult{Module: ModuleSources, Total: len(data.LiveSources) + len(data.EPGSources)}

	// Import live sources first
	for _, src := range data.LiveSources {
		oldID := src.ID

		var existing model.LiveSource
		err := model.DB.Where("name = ?", src.Name).First(&existing).Error
		if err == nil {
			// Existing source — check if busy
			if existing.IsSyncing || existing.IsDetecting {
				mr.Skipped++
				mr.Details = append(mr.Details, fmt.Sprintf("LiveSource \"%s\": syncing/detecting, skipped", src.Name))
				liveIDMap[oldID] = existing.ID
				liveNameMap[src.Name] = existing.ID
				continue
			}
			// Overwrite: remove old scheduler tasks first
			s.scheduler.RemoveLiveSourceTask(existing.ID)
			s.scheduler.RemoveDetectTask(existing.ID)

			// Update fields (preserve ID, timestamps)
			updates := map[string]interface{}{
				"description":     src.Description,
				"type":            src.Type,
				"url":             src.URL,
				"content":         src.Content,
				"headers":         src.Headers,
				"cron_time":       migrateScheduleField(src.CronTime),
				"cron_detect":     migrateScheduleField(src.CronDetect),
				"detect_strategy": src.DetectStrategy,
				"status":          src.Status,
				"iptv_config":     src.IPTVConfig,
			}
			model.DB.Model(&existing).Updates(updates)
			liveIDMap[oldID] = existing.ID
			liveNameMap[src.Name] = existing.ID

			// Re-register scheduler tasks with migrated data
			model.DB.First(&existing, existing.ID) // reload to get migrated values
			s.registerLiveSourceTasks(existing.ID, existing)
			mr.Success++
		} else {
			// New source
			newSrc := src
			newSrc.ID = 0
			newSrc.IsSyncing = false
			newSrc.IsDetecting = false
			newSrc.LastFetchedAt = nil
			newSrc.LastError = ""
			newSrc.CronTime = migrateScheduleField(newSrc.CronTime)
			newSrc.CronDetect = migrateScheduleField(newSrc.CronDetect)
			if err := model.DB.Create(&newSrc).Error; err != nil {
				mr.Failed++
				mr.Details = append(mr.Details, fmt.Sprintf("LiveSource \"%s\": create failed: %s", src.Name, err.Error()))
				continue
			}
			liveIDMap[oldID] = newSrc.ID
			liveNameMap[src.Name] = newSrc.ID
			s.registerLiveSourceTasks(newSrc.ID, src)
			mr.Success++
		}
	}

	// Import EPG sources
	for _, src := range data.EPGSources {
		oldID := src.ID

		// Remap LiveSourceID if present
		if src.LiveSourceID != nil {
			if newLiveID, ok := liveIDMap[*src.LiveSourceID]; ok {
				src.LiveSourceID = &newLiveID
			} else {
				// Try to find by searching existing live sources by name
				// The linked live source might have been imported or already exists
				src.LiveSourceID = nil // Will be nil if cannot resolve
			}
		}

		var existing model.EPGSource
		err := model.DB.Where("name = ?", src.Name).First(&existing).Error
		if err == nil {
			// Check if busy
			if existing.IsSyncing {
				mr.Skipped++
				mr.Details = append(mr.Details, fmt.Sprintf("EPGSource \"%s\": syncing, skipped", src.Name))
				epgIDMap[oldID] = existing.ID
				continue
			}
			// Remove old scheduler
			s.scheduler.RemoveEPGSourceTask(existing.ID)

			updates := map[string]interface{}{
				"description":    src.Description,
				"type":           src.Type,
				"url":            src.URL,
				"headers":        src.Headers,
				"live_source_id": src.LiveSourceID,
				"cron_time":      migrateScheduleField(src.CronTime),
				"status":         src.Status,
				"iptv_config":    src.IPTVConfig,
			}
			model.DB.Model(&existing).Updates(updates)
			epgIDMap[oldID] = existing.ID

			// Re-register scheduler with migrated data
			model.DB.First(&existing, existing.ID) // reload to get migrated values
			if existing.CronTime != "" && src.Status {
				if cfg := parseScheduleConfig(existing.CronTime); cfg != nil {
					s.scheduler.AddEPGSourceTask(existing.ID, cfg)
				}
			}
			mr.Success++
		} else {
			newSrc := src
			newSrc.ID = 0
			newSrc.IsSyncing = false
			newSrc.LastFetchedAt = nil
			newSrc.LastError = ""
			newSrc.CronTime = migrateScheduleField(newSrc.CronTime)
			if err := model.DB.Create(&newSrc).Error; err != nil {
				mr.Failed++
				mr.Details = append(mr.Details, fmt.Sprintf("EPGSource \"%s\": create failed: %s", src.Name, err.Error()))
				continue
			}
			epgIDMap[oldID] = newSrc.ID
			if newSrc.CronTime != "" && newSrc.Status {
				if cfg := parseScheduleConfig(newSrc.CronTime); cfg != nil {
					s.scheduler.AddEPGSourceTask(newSrc.ID, cfg)
				}
			}
			mr.Success++
		}
	}

	return mr
}

func (s *ConfigTransferService) registerLiveSourceTasks(id uint, src model.LiveSource) {
	if src.CronTime != "" && src.Type != model.LiveSourceTypeNetworkManual && src.Status {
		if cfg := parseScheduleConfig(src.CronTime); cfg != nil {
			if err := s.scheduler.AddLiveSourceTask(id, cfg); err != nil {
				slog.Warn("Import: failed to schedule live source task", "id", id, "error", err)
			}
		}
	}
	if src.CronDetect != "" && src.Status {
		if cfg := parseScheduleConfig(src.CronDetect); cfg != nil {
			if err := s.scheduler.AddDetectTask(id, cfg, src.DetectStrategy); err != nil {
				slog.Warn("Import: failed to schedule detect task", "id", id, "error", err)
			}
		}
	}
}

// parseScheduleConfig is a local helper to parse a JSON schedule config string.
// It also handles old-format interval strings (e.g. "6h") by migrating them first.
func parseScheduleConfig(jsonStr string) *model.ScheduleConfig {
	if jsonStr == "" {
		return nil
	}
	// Migrate old format (e.g. "6h") to new JSON format
	migrated := model.MigrateOldInterval(jsonStr)
	if migrated == "" {
		return nil
	}
	var cfg model.ScheduleConfig
	if err := json.Unmarshal([]byte(migrated), &cfg); err != nil {
		return nil
	}
	if cfg.Mode == "" {
		return nil
	}
	return &cfg
}

// migrateScheduleField converts old-format interval strings inline before DB storage.
func migrateScheduleField(val string) string {
	return model.MigrateOldInterval(val)
}

func (s *ConfigTransferService) importLogos(data *ImportParsedData) ModuleResult {
	mr := ModuleResult{Module: ModuleLogos, Total: len(data.Logos)}

	// Ensure logo directory exists
	if err := os.MkdirAll(s.logoDir, 0755); err != nil {
		mr.Failed = len(data.Logos)
		mr.Details = append(mr.Details, fmt.Sprintf("create logo directory failed: %s", err.Error()))
		return mr
	}

	for _, logo := range data.Logos {
		// Use FileName from the export struct (FilePath has json:"-" in the model)
		if logo.FileName == "" {
			mr.Failed++
			mr.Details = append(mr.Details, fmt.Sprintf("Logo \"%s\": missing file name in export data", logo.Name))
			continue
		}
		newFilePath := filepath.Join(s.logoDir, logo.FileName)
		newURLPath := "/logo/" + logo.FileName

		// Look up file data from ZIP
		fileData, hasFile := data.LogoFiles[logo.FileName]
		if !hasFile {
			mr.Failed++
			mr.Details = append(mr.Details, fmt.Sprintf("Logo \"%s\": file not found in ZIP", logo.Name))
			continue
		}

		var existing model.ChannelLogo
		err := model.DB.Where("name = ?", logo.Name).First(&existing).Error
		if err == nil {
			// Overwrite: remove old file
			os.Remove(existing.FilePath)

			// Write new file
			if err := os.WriteFile(newFilePath, fileData, 0644); err != nil {
				mr.Failed++
				mr.Details = append(mr.Details, fmt.Sprintf("Logo \"%s\": write file failed: %s", logo.Name, err.Error()))
				continue
			}

			// Update DB record with new paths
			model.DB.Model(&existing).Updates(map[string]interface{}{
				"file_path": newFilePath,
				"url_path":  newURLPath,
			})
			mr.Success++
		} else {
			// Write new file
			if err := os.WriteFile(newFilePath, fileData, 0644); err != nil {
				mr.Failed++
				mr.Details = append(mr.Details, fmt.Sprintf("Logo \"%s\": write file failed: %s", logo.Name, err.Error()))
				continue
			}

			newLogo := model.ChannelLogo{
				Name:     logo.Name,
				FilePath: newFilePath,
				URLPath:  newURLPath,
			}
			if err := model.DB.Create(&newLogo).Error; err != nil {
				os.Remove(newFilePath)
				mr.Failed++
				mr.Details = append(mr.Details, fmt.Sprintf("Logo \"%s\": create failed: %s", logo.Name, err.Error()))
				continue
			}
			mr.Success++
		}
	}

	return mr
}

func (s *ConfigTransferService) importRules(data *ImportParsedData, ruleIDMap map[uint]uint) ModuleResult {
	mr := ModuleResult{Module: ModuleRules, Total: len(data.Rules)}

	for _, rule := range data.Rules {
		oldID := rule.ID

		var existing model.AggregationRule
		err := model.DB.Where("name = ?", rule.Name).First(&existing).Error
		if err == nil {
			// Overwrite
			updates := map[string]interface{}{
				"description": rule.Description,
				"type":        rule.Type,
				"config":      rule.Config,
				"status":      rule.Status,
			}
			model.DB.Model(&existing).Updates(updates)
			ruleIDMap[oldID] = existing.ID
			mr.Success++
		} else {
			newRule := rule
			newRule.ID = 0
			if err := model.DB.Create(&newRule).Error; err != nil {
				mr.Failed++
				mr.Details = append(mr.Details, fmt.Sprintf("Rule \"%s\": create failed: %s", rule.Name, err.Error()))
				continue
			}
			ruleIDMap[oldID] = newRule.ID
			mr.Success++
		}
	}

	return mr
}

func (s *ConfigTransferService) importPublish(data *ImportParsedData, liveIDMap, epgIDMap, ruleIDMap map[uint]uint) ModuleResult {
	mr := ModuleResult{Module: ModulePublish, Total: len(data.PublishIfaces)}

	for _, iface := range data.PublishIfaces {
		// Remap SourceIDs
		if iface.Type == "live" {
			iface.SourceIDs = remapIDList(iface.SourceIDs, liveIDMap)
			iface.FilterInvalidSourceIDs = remapIDList(iface.FilterInvalidSourceIDs, liveIDMap)
			iface.SourceOutputConfigs = remapJSONObjectKeys(iface.SourceOutputConfigs, liveIDMap)
		} else if iface.Type == "epg" {
			iface.SourceIDs = remapIDList(iface.SourceIDs, epgIDMap)
		}

		// Remap RuleIDs
		iface.RuleIDs = remapIDList(iface.RuleIDs, ruleIDMap)

		var existing model.PublishInterface
		err := model.DB.Where("name = ?", iface.Name).First(&existing).Error
		if err == nil {
			// Overwrite
			updates := map[string]interface{}{
				"description":               iface.Description,
				"path":                      iface.Path,
				"type":                      iface.Type,
				"format":                    iface.Format,
				"source_ids":                iface.SourceIDs,
				"rule_ids":                  iface.RuleIDs,
				"tvg_id_mode":               iface.TvgIDMode,
				"status":                    iface.Status,
				"epg_days":                  iface.EPGDays,
				"gzip_enabled":              iface.GzipEnabled,
				"address_type":              iface.AddressType,
				"multicast_type":            iface.MulticastType,
				"udpxy_url":                 iface.UDPxyURL,
				"fcc_enabled":               iface.FCCEnabled,
				"fcc_type":                  iface.FCCType,
				"custom_params":             iface.CustomParams,
				"m3u_catchup_template":      iface.M3UCatchupTemplate,
				"filter_invalid_source_ids": iface.FilterInvalidSourceIDs,
				"source_output_configs":     iface.SourceOutputConfigs,
				"ua_check_enabled":          iface.UACheckEnabled,
				"ua_allowed_values":         iface.UAAllowedValues,
			}
			if err := model.DB.Model(&existing).Updates(updates).Error; err != nil {
				mr.Failed++
				mr.Details = append(mr.Details, fmt.Sprintf("Publish \"%s\": update failed: %s", iface.Name, err.Error()))
				continue
			}
			mr.Success++
		} else {
			newIface := iface
			newIface.ID = 0
			if err := model.DB.Create(&newIface).Error; err != nil {
				mr.Failed++
				mr.Details = append(mr.Details, fmt.Sprintf("Publish \"%s\": create failed: %s", iface.Name, err.Error()))
				continue
			}
			mr.Success++
		}
	}

	return mr
}

func (s *ConfigTransferService) importDetect(data *ImportParsedData) ModuleResult {
	mr := ModuleResult{Module: ModuleDetect, Total: len(data.DetectSettings)}

	for key, value := range data.DetectSettings {
		var setting model.SystemSetting
		result := model.DB.Where("key = ?", key).First(&setting)
		if result.Error != nil {
			model.DB.Create(&model.SystemSetting{Key: key, Value: value})
		} else {
			model.DB.Model(&setting).Update("value", value)
		}
		mr.Success++
	}

	return mr
}

func (s *ConfigTransferService) importAccessControl(data *ImportParsedData) ModuleResult {
	mr := ModuleResult{Module: ModuleAccessControl, Total: len(data.ACLEntries) + 1}

	// Update mode
	var setting model.SystemSetting
	result := model.DB.Where("key = ?", "access_control_mode").First(&setting)
	if result.Error != nil {
		model.DB.Create(&model.SystemSetting{Key: "access_control_mode", Value: data.ACLMode})
	} else {
		model.DB.Model(&setting).Update("value", data.ACLMode)
	}
	mr.Success++

	// Replace all entries
	model.DB.Where("1 = 1").Delete(&model.AccessControlEntry{})
	for _, entry := range data.ACLEntries {
		newEntry := entry
		newEntry.ID = 0
		if err := model.DB.Create(&newEntry).Error; err != nil {
			mr.Failed++
			mr.Details = append(mr.Details, fmt.Sprintf("ACL entry \"%s\": create failed: %s", entry.Value, err.Error()))
			continue
		}
		mr.Success++
	}

	return mr
}

// ============================================================================
// Utility functions
// ============================================================================

// writeJSON serialises v as JSON and writes it to the zip under the given path.
func writeJSON(zw *zip.Writer, path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fw, err := zw.Create(path)
	if err != nil {
		return err
	}
	_, err = fw.Write(data)
	return err
}

// readZipJSON opens a zip file entry and decodes its JSON into dst.
func readZipJSON(f *zip.File, dst interface{}) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return json.NewDecoder(rc).Decode(dst)
}

// readZipRaw reads an entire zip entry into memory.
func readZipRaw(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// remapIDList takes a comma-separated list of IDs and replaces each using the provided mapping.
// IDs not found in the map are dropped silently.
func remapIDList(idList string, idMap map[uint]uint) string {
	if idList == "" {
		return ""
	}
	parts := strings.Split(idList, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		oldID, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			continue
		}
		if newID, ok := idMap[uint(oldID)]; ok {
			result = append(result, strconv.FormatUint(uint64(newID), 10))
		}
	}
	return strings.Join(result, ",")
}

// remapJSONObjectKeys takes a JSON string like {"1": {...}, "3": {...}} and
// replaces the top-level keys using the provided ID mapping.
func remapJSONObjectKeys(jsonStr string, idMap map[uint]uint) string {
	if jsonStr == "" {
		return ""
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return jsonStr // Return as-is if not valid JSON object
	}

	newObj := make(map[string]json.RawMessage)
	for key, val := range obj {
		oldID, err := strconv.ParseUint(key, 10, 32)
		if err != nil {
			newObj[key] = val // Keep non-numeric keys
			continue
		}
		if newID, ok := idMap[uint(oldID)]; ok {
			newObj[strconv.FormatUint(uint64(newID), 10)] = val
		}
		// Drop unmapped keys
	}

	data, err := json.Marshal(newObj)
	if err != nil {
		return jsonStr
	}
	return string(data)
}
