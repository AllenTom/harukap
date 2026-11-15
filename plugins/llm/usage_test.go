package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 配额是"无人值守烧掉 token"的最后一道闸，这几条必须钉住。

func TestUsageQuotaBlocksAtLimit(t *testing.T) {
	u := newUsageTracker()
	for i := 0; i < 3; i++ {
		if err := u.acquire(3); err != nil {
			t.Fatalf("第 %d 次不该被拦: %v", i+1, err)
		}
	}
	err := u.acquire(3)
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("超过上限应返回 ErrQuotaExceeded，得到 %v", err)
	}
	if u.calls != 3 {
		t.Fatalf("被拒绝的请求不该计数，calls=%d", u.calls)
	}
}

func TestUsageLimitZeroMeansUnlimited(t *testing.T) {
	u := newUsageTracker()
	for i := 0; i < 100; i++ {
		if err := u.acquire(0); err != nil {
			t.Fatalf("limit=0 应表示不限，第 %d 次失败: %v", i+1, err)
		}
	}
	if snap := u.snapshot(0); snap.Remaining != -1 {
		t.Fatalf("不限时 Remaining 应为 -1，得到 %d", snap.Remaining)
	}
}

func TestUsageReleaseRollsBack(t *testing.T) {
	u := newUsageTracker()
	if err := u.acquire(1); err != nil {
		t.Fatal(err)
	}
	u.release()
	if u.calls != 0 {
		t.Fatalf("release 后 calls 应为 0，得到 %d", u.calls)
	}
	if err := u.acquire(1); err != nil {
		t.Fatalf("退还后应重新可用: %v", err)
	}
}

func TestUsageRolloverOnNewDay(t *testing.T) {
	u := newUsageTracker()
	_ = u.acquire(10)
	// 模拟昨天用过 1 次
	u.day = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if snap := u.snapshot(10); snap.Calls != 0 {
		t.Fatalf("跨天后应清零，calls=%d", snap.Calls)
	}
	if u.day != today() {
		t.Fatalf("rollover 应把 day 更新为今天，得到 %s", u.day)
	}
}

func TestUsageSnapshotBuckets(t *testing.T) {
	u := newUsageTracker()
	_ = u.acquire(10)
	_ = u.acquire(10)
	u.record(SceneComicTagging, "deepseek", true)
	u.record(SceneComicTagging, "deepseek", false)
	u.record(ScenePhotoSearchRewrite, "deepseek", true)

	snap := u.snapshot(10)
	if snap.Calls != 2 || snap.Failed != 1 {
		t.Fatalf("calls/failed 统计不对: %+v", snap)
	}
	if snap.ByScene[SceneComicTagging] != 2 || snap.ByScene[ScenePhotoSearchRewrite] != 1 {
		t.Fatalf("按场景分桶不对: %+v", snap.ByScene)
	}
	if snap.ByProvider["deepseek"] != 3 {
		t.Fatalf("按提供商分桶不对: %+v", snap.ByProvider)
	}
	if snap.Remaining != 8 {
		t.Fatalf("remaining 应为 8，得到 %d", snap.Remaining)
	}
}

// generateWithScene 是唯一出口：超时要兜底，配额要真的拦住。
func TestGenerateWithSceneEnforcesQuota(t *testing.T) {
	p := NewPlugin()
	p.Config = &LLMConfig{Enable: true, Default: "fake", DailyCallLimit: 1}
	p.providers = map[string]LLMProvider{"fake": &fakeProvider{}}

	if _, err := p.GenerateTextForScene(context.Background(), SceneComicTagging, "hi"); err != nil {
		t.Fatalf("第一次调用不该失败: %v", err)
	}
	_, err := p.GenerateTextForScene(context.Background(), SceneComicTagging, "hi")
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("第二次应被配额拦住，得到 %v", err)
	}
	if snap := p.usage.snapshot(1); snap.Calls != 1 {
		t.Fatalf("只应有 1 次计入，得到 %d", snap.Calls)
	}
}

func TestGenerateWithSceneAppliesDefaultTimeout(t *testing.T) {
	p := NewPlugin()
	p.Config = &LLMConfig{Enable: true}
	probe := &fakeProvider{}
	p.providers = map[string]LLMProvider{"fake": probe}
	p.Config.Default = "fake"

	if _, err := p.GenerateTextForScene(context.Background(), "", "hi"); err != nil {
		t.Fatalf("调用失败: %v", err)
	}
	if !probe.sawDeadline {
		t.Fatal("调用方没给 deadline 时，generateWithScene 应补一个默认超时")
	}
}

type fakeProvider struct {
	sawDeadline bool
}

func (f *fakeProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	_, f.sawDeadline = ctx.Deadline()
	return "ok", nil
}

func (f *fakeProvider) GetProviderName() string { return "fake" }
