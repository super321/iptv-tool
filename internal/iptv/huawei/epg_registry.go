package huawei

import (
	"iptv-tool-v2/internal/iptv"
	"sync"
)

var (
	epgStrategiesMu  sync.RWMutex
	epgStrategies    = make(map[string]iptv.EPGFetchStrategy)
	epgStrategyOrder []string // Keep an ordered slice for deterministic auto-detection
)

// RegisterEPGStrategy registers a new EPG fetch strategy.
// This should be called in the init() function of each strategy implementation file.
func RegisterEPGStrategy(strategy iptv.EPGFetchStrategy) {
	epgStrategiesMu.Lock()
	defer epgStrategiesMu.Unlock()

	name := strategy.Name()
	if _, exists := epgStrategies[name]; !exists {
		epgStrategyOrder = append(epgStrategyOrder, name)
	}
	epgStrategies[name] = strategy
}

// GetEPGStrategy retrieves a registered strategy by name.
func GetEPGStrategy(name string) iptv.EPGFetchStrategy {
	epgStrategiesMu.RLock()
	defer epgStrategiesMu.RUnlock()
	return epgStrategies[name]
}

// GetAllEPGStrategies returns all registered strategies in the order they were registered.
func GetAllEPGStrategies() []iptv.EPGFetchStrategy {
	epgStrategiesMu.RLock()
	defer epgStrategiesMu.RUnlock()

	strategies := make([]iptv.EPGFetchStrategy, 0, len(epgStrategyOrder))
	for _, name := range epgStrategyOrder {
		strategies = append(strategies, epgStrategies[name])
	}
	return strategies
}
