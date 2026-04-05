package publish

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLiveCache_SetAndGet(t *testing.T) {
	InvalidateAll()

	channels := []AggregatedChannel{{Name: "CCTV-1"}}
	setLiveChannels(1, channels)

	got, ok := getLiveChannels(1)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got) != 1 || got[0].Name != "CCTV-1" {
		t.Errorf("got %v, want [{Name:CCTV-1}]", got)
	}
}

func TestLiveCache_Miss(t *testing.T) {
	InvalidateAll()
	_, ok := getLiveChannels(999)
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestLiveCache_TTLExpiration(t *testing.T) {
	InvalidateAll()
	origTTL := cacheTTL
	cacheTTL = 10 * time.Millisecond
	defer func() { cacheTTL = origTTL }()

	setLiveChannels(1, []AggregatedChannel{{Name: "test"}})
	if _, ok := getLiveChannels(1); !ok {
		t.Error("expected hit immediately after set")
	}
	time.Sleep(20 * time.Millisecond)
	if _, ok := getLiveChannels(1); ok {
		t.Error("expected miss after TTL expiration")
	}
}

func TestEPGCache_SetAndGet(t *testing.T) {
	InvalidateAll()
	epg := &AggregatedEPG{
		Channels:     make(map[string]*EPGChannelPrograms),
		ChannelOrder: []string{"ch1"},
	}
	setEPGPrograms(1, epg)

	got, ok := getEPGPrograms(1)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got.ChannelOrder) != 1 || got.ChannelOrder[0] != "ch1" {
		t.Errorf("unexpected: %v", got.ChannelOrder)
	}
}

func TestInvalidateAll(t *testing.T) {
	setLiveChannels(1, []AggregatedChannel{{Name: "test"}})
	setEPGPrograms(1, &AggregatedEPG{})
	InvalidateAll()
	if _, ok := getLiveChannels(1); ok {
		t.Error("live cache should be empty")
	}
	if _, ok := getEPGPrograms(1); ok {
		t.Error("epg cache should be empty")
	}
}

func TestLoadOrStoreLiveChannels_LoaderCalledOnce(t *testing.T) {
	InvalidateAll()
	var callCount atomic.Int32
	loader := func() ([]AggregatedChannel, error) {
		callCount.Add(1)
		return []AggregatedChannel{{Name: "loaded"}}, nil
	}

	ch1, err := LoadOrStoreLiveChannels(10, loader)
	if err != nil {
		t.Fatal(err)
	}
	if len(ch1) != 1 || ch1[0].Name != "loaded" {
		t.Errorf("unexpected: %v", ch1)
	}

	_, _ = LoadOrStoreLiveChannels(10, loader)
	if callCount.Load() != 1 {
		t.Errorf("loader called %d times, want 1", callCount.Load())
	}
}

func TestLoadOrStoreLiveChannels_LoaderError(t *testing.T) {
	InvalidateAll()
	loader := func() ([]AggregatedChannel, error) {
		return nil, errors.New("db error")
	}
	_, err := LoadOrStoreLiveChannels(20, loader)
	if err == nil {
		t.Error("expected error from loader")
	}
	if _, ok := getLiveChannels(20); ok {
		t.Error("cache should not be set on error")
	}
}

func TestLoadOrStoreEPGPrograms_LoaderCalledOnce(t *testing.T) {
	InvalidateAll()
	var callCount atomic.Int32
	loader := func() (*AggregatedEPG, error) {
		callCount.Add(1)
		return &AggregatedEPG{ChannelOrder: []string{"ch"}}, nil
	}

	epg1, err := LoadOrStoreEPGPrograms(10, loader)
	if err != nil {
		t.Fatal(err)
	}
	if len(epg1.ChannelOrder) != 1 {
		t.Error("unexpected result")
	}

	_, _ = LoadOrStoreEPGPrograms(10, loader)
	if callCount.Load() != 1 {
		t.Errorf("loader called %d times, want 1", callCount.Load())
	}
}

func TestLoadOrStoreLiveChannels_Concurrent(t *testing.T) {
	InvalidateAll()
	var callCount atomic.Int32
	loader := func() ([]AggregatedChannel, error) {
		callCount.Add(1)
		time.Sleep(50 * time.Millisecond)
		return []AggregatedChannel{{Name: "result"}}, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, err := LoadOrStoreLiveChannels(30, loader)
			if err != nil {
				t.Errorf("error: %v", err)
				return
			}
			if len(ch) != 1 {
				t.Errorf("len = %d", len(ch))
			}
		}()
	}
	wg.Wait()

	if callCount.Load() != 1 {
		t.Errorf("loader called %d times, want 1 (stampede prevention)", callCount.Load())
	}
}
