package api

import (
	"testing"
	"time"
)

func TestIPRateLimiter_AllowUnderLimit(t *testing.T) {
	rl := newIPRateLimiter()
	for i := 0; i < rateLimitMax; i++ {
		if !rl.Allow("192.168.1.1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

func TestIPRateLimiter_RejectOverLimit(t *testing.T) {
	rl := newIPRateLimiter()
	for i := 0; i < rateLimitMax; i++ {
		rl.Allow("192.168.1.1")
	}
	if rl.Allow("192.168.1.1") {
		t.Error("should be rejected over limit")
	}
}

func TestIPRateLimiter_DifferentIPsIndependent(t *testing.T) {
	rl := newIPRateLimiter()
	for i := 0; i < rateLimitMax; i++ {
		rl.Allow("192.168.1.1")
	}
	if !rl.Allow("192.168.1.2") {
		t.Error("different IP should not be affected")
	}
}

func TestIPRateLimiter_Cleanup(t *testing.T) {
	rl := newIPRateLimiter()
	rl.mu.Lock()
	rl.records["old-ip"] = []time.Time{time.Now().Add(-cleanupExpiration - time.Minute)}
	rl.records["recent-ip"] = []time.Time{time.Now()}
	rl.mu.Unlock()

	rl.cleanup()

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if _, ok := rl.records["old-ip"]; ok {
		t.Error("old-ip should be cleaned up")
	}
	if _, ok := rl.records["recent-ip"]; !ok {
		t.Error("recent-ip should remain")
	}
}

func TestIPRateLimiter_CleanupEmpty(t *testing.T) {
	rl := newIPRateLimiter()
	rl.mu.Lock()
	rl.records["empty-ip"] = []time.Time{}
	rl.mu.Unlock()
	rl.cleanup()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if _, ok := rl.records["empty-ip"]; ok {
		t.Error("empty record should be cleaned up")
	}
}

func TestIPRateLimiter_RecordsCapped(t *testing.T) {
	rl := newIPRateLimiter()
	for i := 0; i < rateLimitMax+5; i++ {
		rl.Allow("10.0.0.1")
	}
	rl.mu.Lock()
	count := len(rl.records["10.0.0.1"])
	rl.mu.Unlock()
	if count > rateLimitMax {
		t.Errorf("records = %d, should be capped at %d", count, rateLimitMax)
	}
}

func TestLoginAttemptTracker_RecordFailure(t *testing.T) {
	tr := newLoginAttemptTracker()
	if c := tr.RecordFailure("admin"); c != 1 {
		t.Errorf("first = %d, want 1", c)
	}
	if c := tr.RecordFailure("admin"); c != 2 {
		t.Errorf("second = %d, want 2", c)
	}
}

func TestLoginAttemptTracker_NeedCaptcha(t *testing.T) {
	tr := newLoginAttemptTracker()
	for i := 0; i < failedThreshold-1; i++ {
		tr.RecordFailure("admin")
	}
	if tr.NeedCaptcha("admin") {
		t.Error("below threshold should not need captcha")
	}
	tr.RecordFailure("admin")
	if !tr.NeedCaptcha("admin") {
		t.Error("at threshold should need captcha")
	}
}

func TestLoginAttemptTracker_UnknownUser(t *testing.T) {
	tr := newLoginAttemptTracker()
	if tr.NeedCaptcha("unknown") {
		t.Error("unknown user should not need captcha")
	}
}

func TestLoginAttemptTracker_Reset(t *testing.T) {
	tr := newLoginAttemptTracker()
	for i := 0; i < failedThreshold; i++ {
		tr.RecordFailure("admin")
	}
	tr.Reset("admin")
	if tr.NeedCaptcha("admin") {
		t.Error("should not need captcha after reset")
	}
}

func TestLoginAttemptTracker_IndependentUsers(t *testing.T) {
	tr := newLoginAttemptTracker()
	for i := 0; i < failedThreshold; i++ {
		tr.RecordFailure("user1")
	}
	if tr.NeedCaptcha("user2") {
		t.Error("user2 should not be affected by user1")
	}
}

func TestLoginAttemptTracker_Cleanup(t *testing.T) {
	tr := newLoginAttemptTracker()
	tr.mu.Lock()
	tr.records["old"] = &attemptRecord{failures: 5, lastEntry: time.Now().Add(-cleanupExpiration - time.Minute)}
	tr.records["new"] = &attemptRecord{failures: 2, lastEntry: time.Now()}
	tr.mu.Unlock()

	tr.cleanup()

	tr.mu.Lock()
	defer tr.mu.Unlock()
	if _, ok := tr.records["old"]; ok {
		t.Error("old record should be cleaned up")
	}
	if _, ok := tr.records["new"]; !ok {
		t.Error("new record should remain")
	}
}
