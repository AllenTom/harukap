package llm

import (
	"strings"
	"testing"
)

// 预设表是生成出来的数据，最容易犯的错是"重复 id / kind 拼错 / 忘填地址"，
// 这类错误在 UI 上表现为"选了模板却没反应"，很难从现象反推。这里一次钉住。
func TestProviderPresetsSanity(t *testing.T) {
	presets := ProviderPresets()
	if len(presets) < 20 {
		t.Fatalf("预设数量过少：%d，生成脚本可能没跑", len(presets))
	}

	seen := make(map[string]bool, len(presets))
	for _, p := range presets {
		if p.ID == "" || p.Label == "" {
			t.Errorf("预设缺少 id 或 label: %+v", p)
		}
		if seen[p.ID] {
			t.Errorf("预设 id 重复: %s", p.ID)
		}
		seen[p.ID] = true

		if !p.Kind.valid() {
			t.Errorf("%s: 非法 kind %q", p.ID, p.Kind)
		}
		if p.Region == "" {
			t.Errorf("%s: 缺少 region", p.ID)
		}
		// gemini / vertex 的地址由 SDK 决定，本地 ollama 有默认值，其余必须写明
		if p.BaseURL == "" && p.Kind != KindGemini {
			t.Errorf("%s: 缺少 base_url", p.ID)
		}
		if p.BaseURL != "" && strings.HasSuffix(p.BaseURL, "/") {
			t.Errorf("%s: base_url 不该以 / 结尾（会拼出 //chat/completions）: %s", p.ID, p.BaseURL)
		}
		if p.DefaultModel != "" && len(p.Models) > 0 {
			found := false
			for _, m := range p.Models {
				if m.ID == p.DefaultModel {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: 默认模型 %s 不在候选列表里", p.ID, p.DefaultModel)
			}
		}
		for _, m := range p.Models {
			if m.ID == "" {
				t.Errorf("%s: 存在空模型 id", p.ID)
			}
			switch m.Tier {
			case "", "fast", "balanced", "quality":
			default:
				t.Errorf("%s/%s: 未知档位 %q", p.ID, m.ID, m.Tier)
			}
		}
	}
}

// Anthropic 的 base_url 有两种写法（带 /v1 与不带），拼错就是 404，
// 这里把两种都钉住。
func TestAnthropicEndpoint(t *testing.T) {
	cases := map[string]string{
		"https://api.anthropic.com":              "https://api.anthropic.com/v1/messages",
		"https://api.anthropic.com/":             "https://api.anthropic.com/v1/messages",
		"https://api.minimax.cn/anthropic/v1":    "https://api.minimax.cn/anthropic/v1/messages",
		"https://open.bigmodel.cn/api/anthropic": "https://open.bigmodel.cn/api/anthropic/v1/messages",
	}
	for base, want := range cases {
		p := newAnthropicProvider(&ProviderSpec{ID: "t", Kind: KindAnthropic, BaseURL: base})
		if got := p.endpoint("/messages"); got != want {
			t.Errorf("base=%s: got %s, want %s", base, got, want)
		}
	}
}
