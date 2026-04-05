package task

import (
	"testing"

	"iptv-tool-v2/internal/model"
)

func TestParseScheduleConfig_Valid(t *testing.T) {
	input := `{"mode":"interval","hours":6}`
	cfg, err := ParseScheduleConfig(input)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Mode != "interval" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "interval")
	}
	if cfg.Hours != 6 {
		t.Errorf("Hours = %d, want 6", cfg.Hours)
	}
}

func TestParseScheduleConfig_Daily(t *testing.T) {
	input := `{"mode":"daily","days":2,"times":["08:00","20:00"]}`
	cfg, err := ParseScheduleConfig(input)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != "daily" {
		t.Errorf("Mode = %q", cfg.Mode)
	}
	if cfg.Days != 2 {
		t.Errorf("Days = %d", cfg.Days)
	}
	if len(cfg.Times) != 2 {
		t.Fatalf("Times len = %d, want 2", len(cfg.Times))
	}
}

func TestParseScheduleConfig_Empty(t *testing.T) {
	cfg, err := ParseScheduleConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Error("empty input should return nil")
	}
}

func TestParseScheduleConfig_Whitespace(t *testing.T) {
	cfg, err := ParseScheduleConfig("   ")
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Error("whitespace only should return nil")
	}
}

func TestParseScheduleConfig_InvalidJSON(t *testing.T) {
	_, err := ParseScheduleConfig("{bad json}")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestMarshalScheduleConfig_Nil(t *testing.T) {
	result := MarshalScheduleConfig(nil)
	if result != "" {
		t.Errorf("nil config should return empty, got %q", result)
	}
}

func TestMarshalScheduleConfig_Empty(t *testing.T) {
	cfg := &ScheduleConfig{}
	result := MarshalScheduleConfig(cfg)
	if result != "" {
		t.Errorf("empty config should return empty, got %q", result)
	}
}

func TestMarshalScheduleConfig_RoundTrip(t *testing.T) {
	original := &ScheduleConfig{Mode: "interval", Hours: 12}
	marshaled := MarshalScheduleConfig(original)
	if marshaled == "" {
		t.Fatal("marshal returned empty")
	}
	parsed, err := ParseScheduleConfig(marshaled)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Mode != original.Mode || parsed.Hours != original.Hours {
		t.Errorf("round trip mismatch: got {%s, %d}", parsed.Mode, parsed.Hours)
	}
}

func TestValidateScheduleConfig_Nil(t *testing.T) {
	if err := ValidateScheduleConfig(nil, "en"); err != nil {
		t.Errorf("nil config should be valid: %v", err)
	}
}

func TestValidateScheduleConfig_EmptyMode(t *testing.T) {
	cfg := &ScheduleConfig{}
	if err := ValidateScheduleConfig(cfg, "en"); err != nil {
		t.Errorf("empty mode should be valid: %v", err)
	}
}

func TestValidateScheduleConfig_IntervalValid(t *testing.T) {
	cfg := &ScheduleConfig{Mode: model.ScheduleModeInterval, Hours: 6}
	if err := ValidateScheduleConfig(cfg, "en"); err != nil {
		t.Errorf("valid interval should pass: %v", err)
	}
}

func TestValidateScheduleConfig_IntervalTooLow(t *testing.T) {
	cfg := &ScheduleConfig{Mode: model.ScheduleModeInterval, Hours: 0}
	if err := ValidateScheduleConfig(cfg, "en"); err == nil {
		t.Error("hours=0 should fail")
	}
}

func TestValidateScheduleConfig_IntervalTooHigh(t *testing.T) {
	cfg := &ScheduleConfig{Mode: model.ScheduleModeInterval, Hours: 100}
	if err := ValidateScheduleConfig(cfg, "en"); err == nil {
		t.Error("hours=100 should fail")
	}
}

func TestValidateScheduleConfig_DailyValid(t *testing.T) {
	cfg := &ScheduleConfig{Mode: model.ScheduleModeDaily, Days: 1, Times: []string{"08:00"}}
	if err := ValidateScheduleConfig(cfg, "en"); err != nil {
		t.Errorf("valid daily should pass: %v", err)
	}
}

func TestValidateScheduleConfig_DailyDaysTooHigh(t *testing.T) {
	cfg := &ScheduleConfig{Mode: model.ScheduleModeDaily, Days: 31}
	if err := ValidateScheduleConfig(cfg, "en"); err == nil {
		t.Error("days=31 should fail")
	}
}

func TestValidateScheduleConfig_InvalidMode(t *testing.T) {
	cfg := &ScheduleConfig{Mode: "unknown"}
	if err := ValidateScheduleConfig(cfg, "en"); err == nil {
		t.Error("invalid mode should fail")
	}
}

func TestParseTimeToMinutes(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"00:00", 0, false},
		{"08:30", 510, false},
		{"23:59", 1439, false},
		{"12:00", 720, false},
		{"24:00", 0, true},
		{"12:60", 0, true},
		{"-1:00", 0, true},
		{"abc", 0, true},
		{"12", 0, true},
		{"12:00:00", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseTimeToMinutes(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeToMinutes(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseTimeToMinutes(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateTimePoints_Duplicates(t *testing.T) {
	err := validateTimePoints([]string{"08:00", "08:00"})
	if err == nil {
		t.Error("duplicate times should fail")
	}
}

func TestValidateTimePoints_TooClose(t *testing.T) {
	err := validateTimePoints([]string{"08:00", "08:20"})
	if err == nil {
		t.Error("times less than 30 min apart should fail")
	}
}

func TestValidateTimePoints_WrapAround(t *testing.T) {
	// 23:50 and 00:10 → gap = 20 min (< 30)
	err := validateTimePoints([]string{"23:50", "00:10"})
	if err == nil {
		t.Error("wrap-around gap < 30 min should fail")
	}
}

func TestValidateTimePoints_Valid(t *testing.T) {
	err := validateTimePoints([]string{"06:00", "12:00", "18:00"})
	if err != nil {
		t.Errorf("valid time points should pass: %v", err)
	}
}

func TestValidateTimePoints_TooMany(t *testing.T) {
	times := []string{"00:00", "05:00", "10:00", "15:00", "20:00", "23:00"}
	err := validateTimePoints(times)
	if err == nil {
		t.Error("more than MaxTimePoints should fail")
	}
}

func TestValidateTimePoints_Empty(t *testing.T) {
	err := validateTimePoints([]string{})
	if err == nil {
		t.Error("empty times should fail")
	}
}

func TestScheduleConfig_IsEmpty(t *testing.T) {
	var nilCfg *ScheduleConfig
	if !nilCfg.IsEmpty() {
		t.Error("nil config should be empty")
	}
	if !(&ScheduleConfig{}).IsEmpty() {
		t.Error("zero-value config should be empty")
	}
	if (&ScheduleConfig{Mode: "interval"}).IsEmpty() {
		t.Error("config with mode should not be empty")
	}
}

func TestScheduleConfig_String(t *testing.T) {
	tests := []struct {
		name string
		cfg  *ScheduleConfig
		want string
	}{
		{"nil", nil, ""},
		{"empty", &ScheduleConfig{}, ""},
		{"interval", &ScheduleConfig{Mode: "interval", Hours: 6}, "every 6h"},
		{"daily single", &ScheduleConfig{Mode: "daily", Days: 1, Times: []string{"08:00"}}, "daily at 08:00"},
		{"daily multi-day", &ScheduleConfig{Mode: "daily", Days: 3, Times: []string{"08:00", "20:00"}}, "every 3d at 08:00, 20:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
