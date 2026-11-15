package llm

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ErrQuotaExceeded 当日调用次数用尽。调用方应当把它当成"功能暂时不可用"处理，
// 而不是重试——重试只会继续撞墙。
var ErrQuotaExceeded = fmt.Errorf("LLM 今日调用次数已达上限")

// UsageSnapshot 是 /llm/status 暴露给前端的用量视图。
//
// 为什么不记 token 而记调用次数：token 数只有部分上游会在响应里返回，
// OpenAI/Gemini/Ollama 口径不一致，拿到也是估算；调用次数是本地可以精确统计的，
// 而且"上限次数的调用"和"上限 token"在业务上等价——每个场景的 prompt 长度基本恒定。
type UsageSnapshot struct {
	Day        string         `json:"day"`
	Calls      int            `json:"calls"`
	Failed     int            `json:"failed"`
	Limit      int            `json:"limit"`     // 0 表示不限
	Remaining  int            `json:"remaining"` // -1 表示不限
	ByScene    map[string]int `json:"by_scene"`  // 场景 → 次数
	ByProvider map[string]int `json:"by_provider"`
	ResetsAt   string         `json:"resets_at"` // 次日零点（本地时区）
}

// usageTracker 进程内计数器，按自然日重置。
//
// 每个服务是独立进程、独立计数：配额是"单个服务每天最多 N 次"，
// 不是"整套系统共用 N 次"。这样某个服务跑飞不会把别的服务拖死。
type usageTracker struct {
	mu         sync.Mutex
	day        string
	calls      int
	failed     int
	byScene    map[string]int
	byProvider map[string]int
}

func newUsageTracker() *usageTracker {
	return &usageTracker{
		day:        today(),
		byScene:    make(map[string]int),
		byProvider: make(map[string]int),
	}
}

func today() string { return time.Now().Format("2006-01-02") }

// rolloverLocked 跨天清零。调用方必须持锁。
func (t *usageTracker) rolloverLocked() {
	d := today()
	if t.day == d {
		return
	}
	t.day = d
	t.calls = 0
	t.failed = 0
	t.byScene = make(map[string]int)
	t.byProvider = make(map[string]int)
}

// acquire 在真正发起请求前占位。超过当日上限返回 ErrQuotaExceeded。
//
// 先占位再请求，而不是"请求成功后再累加"：并发场景下后者会让 N 个请求
// 同时看到 remaining>0 而一起放行，上限就形同虚设。
func (t *usageTracker) acquire(limit int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rolloverLocked()
	if limit > 0 && t.calls >= limit {
		return fmt.Errorf("%w（%d/%d）", ErrQuotaExceeded, t.calls, limit)
	}
	t.calls++
	return nil
}

// release 请求没能真正发出去（provider 不存在之类的本地错误）时退还占位。
func (t *usageTracker) release() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.calls > 0 {
		t.calls--
	}
}

// record 记录一次已发出的调用。ok=false 也算：上游报错、超时通常同样计费。
func (t *usageTracker) record(scene, provider string, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rolloverLocked()
	if !ok {
		t.failed++
	}
	if scene != "" {
		t.byScene[scene]++
	}
	if provider != "" {
		t.byProvider[provider]++
	}
}

func (t *usageTracker) snapshot(limit int) UsageSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rolloverLocked()

	// 次日零点用本地时区显式计算：Time.Truncate 是按 UTC 零时刻对齐的，
	// 在东八区会算出"今天 08:00"这种误导值。
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	snap := UsageSnapshot{
		Day:        t.day,
		Calls:      t.calls,
		Failed:     t.failed,
		Limit:      limit,
		Remaining:  -1,
		ByScene:    copyCounts(t.byScene),
		ByProvider: copyCounts(t.byProvider),
		ResetsAt:   midnight.Format(time.RFC3339),
	}
	if limit > 0 {
		snap.Remaining = limit - t.calls
		if snap.Remaining < 0 {
			snap.Remaining = 0
		}
	}
	return snap
}

func copyCounts(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out[k] = in[k]
	}
	return out
}
