package api

import (
	"sync"
	"time"
)

// --- IP Rate Limiter ---
// 基于滑动窗口的 IP 频率限制器
// 每个 IP 维护一个时间戳切片，每次请求追加当前时间
// 检查时移除窗口外的记录，剩余数量 >= maxAttempts 则拒绝

const (
	rateLimitWindow   = 1 * time.Minute  // 滑动窗口大小
	rateLimitMax      = 5                // 每个窗口最大请求次数
	failedThreshold   = 3                // 连续失败达到此次数后触发验证码
	cleanupInterval   = 5 * time.Minute  // 定时清理间隔
	cleanupExpiration = 10 * time.Minute // 过期时间，超过此时间无活动则清理
)

type ipRateLimiter struct {
	mu      sync.Mutex
	records map[string][]time.Time
}

func newIPRateLimiter() *ipRateLimiter {
	return &ipRateLimiter{
		records: make(map[string][]time.Time),
	}
}

// Allow 检查指定 IP 是否允许请求
// 返回 true 表示允许，false 表示超出限制
func (rl *ipRateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)

	// 获取该 IP 的请求记录并移除窗口外的旧记录
	timestamps := rl.records[ip]
	valid := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}

	// 超出限制则拒绝
	if len(valid) >= rateLimitMax {
		// 被拒绝的请求也计入记录，确保持续攻击时窗口不断延长
		// 仅保留最近 rateLimitMax 条记录，防止持续攻击导致内存膨胀
		valid = append(valid, now)
		if len(valid) > rateLimitMax {
			valid = valid[len(valid)-rateLimitMax:]
		}
		rl.records[ip] = valid
		return false
	}

	// 记录本次请求
	rl.records[ip] = append(valid, now)
	return true
}

// cleanup 清理过期的 IP 记录
func (rl *ipRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-cleanupExpiration)
	for ip, timestamps := range rl.records {
		if len(timestamps) == 0 || timestamps[len(timestamps)-1].Before(cutoff) {
			delete(rl.records, ip)
		}
	}
}

// --- Login Attempt Tracker ---
// 按用户名追踪连续失败次数
// 登录成功后重置

type attemptRecord struct {
	failures  int
	lastEntry time.Time
}

type loginAttemptTracker struct {
	mu      sync.Mutex
	records map[string]*attemptRecord
}

func newLoginAttemptTracker() *loginAttemptTracker {
	return &loginAttemptTracker{
		records: make(map[string]*attemptRecord),
	}
}

// RecordFailure 记录一次失败并返回累计失败次数
func (t *loginAttemptTracker) RecordFailure(username string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	rec, ok := t.records[username]
	if !ok {
		rec = &attemptRecord{}
		t.records[username] = rec
	}
	rec.failures++
	rec.lastEntry = time.Now()
	return rec.failures
}

// NeedCaptcha 检查该用户名是否需要验证码
func (t *loginAttemptTracker) NeedCaptcha(username string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	rec, ok := t.records[username]
	if !ok {
		return false
	}
	return rec.failures >= failedThreshold
}

// Reset 重置指定用户名的失败计数
func (t *loginAttemptTracker) Reset(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.records, username)
}

// cleanup 清理过期的失败记录
func (t *loginAttemptTracker) cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-cleanupExpiration)
	for username, rec := range t.records {
		if rec.lastEntry.Before(cutoff) {
			delete(t.records, username)
		}
	}
}

// --- 全局实例 ---

var (
	globalRateLimiter    *ipRateLimiter
	globalAttemptTracker *loginAttemptTracker
	cleanupOnce          sync.Once
)

func init() {
	globalRateLimiter = newIPRateLimiter()
	globalAttemptTracker = newLoginAttemptTracker()

	// 启动后台清理协程（仅启动一次）
	cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(cleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				globalRateLimiter.cleanup()
				globalAttemptTracker.cleanup()
			}
		}()
	})
}
