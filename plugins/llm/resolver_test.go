package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempProviders(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "llm-providers.yml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp providers: %v", err)
	}
	return path
}

const sharedYAML = `
version: 1
providers:
  gemini:
    kind: gemini
    enable: true
    api_key: test-key
    default_model: gemini-3.6-flash
  deepseek:
    kind: openai-compatible
    enable: true
    base_url: https://api.deepseek.com/v1
    api_key: sk-test
    default_model: deepseek-chat
    models:
      - id: deepseek-chat
        tier: fast
  openai:
    kind: openai-compatible
    enable: false
    base_url: https://api.openai.com/v1
`

func TestResolveInheritOverrideMissing(t *testing.T) {
	path := writeTempProviders(t, sharedYAML)
	shared, err := LoadProvidersFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	cfg := &LLMConfig{
		Enable:        true,
		Default:       "gemini",
		ProvidersFile: path,
		Providers: map[string]*ProviderOverride{
			// 全继承
			"gemini": {Mode: "inherit"},
			// 本地覆盖：换成自己那台 ollama
			"ollama": {Mode: "override", ProviderSpec: ProviderSpec{
				Kind: KindOllama, Enable: true, BaseURL: "http://192.168.50.9:11434",
				DefaultModel: "qwen2.5",
			}},
			// 全局没有、本地也声明继承 → 必须显式不可用
			"kimi": {Mode: "inherit"},
		},
	}

	specs, sources := ResolveProviderSpecs(cfg, shared)

	if sources["gemini"] != "global" {
		t.Fatalf("gemini 应继承全局，实际 %q", sources["gemini"])
	}
	if ok, reason := specs["gemini"].Ready(); !ok {
		t.Fatalf("gemini 应可用，原因: %s", reason)
	}
	if sources["deepseek"] != "global" {
		t.Fatalf("deepseek 应来自全局，实际 %q", sources["deepseek"])
	}

	if sources["ollama"] != "local" {
		t.Fatalf("ollama 应为本地覆盖，实际 %q", sources["ollama"])
	}
	if specs["ollama"].BaseURL != "http://192.168.50.9:11434" {
		t.Fatalf("override 应整段替换，实际 base_url=%q", specs["ollama"].BaseURL)
	}

	if sources["kimi"] != "missing" {
		t.Fatalf("kimi 应为 missing，实际 %q", sources["kimi"])
	}

	list := DescribeProviders(specs, sources)
	var kimi *ResolvedProvider
	for i := range list {
		if list[i].ID == "kimi" {
			kimi = &list[i]
		}
	}
	if kimi == nil || kimi.Ready || kimi.Reason == "" {
		t.Fatalf("kimi 必须不可用且带原因，实际 %+v", kimi)
	}
}

func TestReadyReasons(t *testing.T) {
	cases := []struct {
		name string
		spec ProviderSpec
		want bool
	}{
		{"openai 缺 key", ProviderSpec{Kind: KindOpenAICompatible, Enable: true, BaseURL: "https://x/v1"}, false},
		{"openai 完整", ProviderSpec{Kind: KindOpenAICompatible, Enable: true, BaseURL: "https://x/v1", APIKey: "k"}, true},
		{"ollama 只有地址", ProviderSpec{Kind: KindOllama, Enable: true, BaseURL: "http://h:11434"}, true},
		{"ollama 缺地址", ProviderSpec{Kind: KindOllama, Enable: true}, false},
		{"gemini 无 key 无 vertex", ProviderSpec{Kind: KindGemini, Enable: true}, false},
		{"gemini vertex", ProviderSpec{Kind: KindGemini, Enable: true, Project: "p", Location: "l"}, true},
		{"已停用", ProviderSpec{Kind: KindOllama, Enable: false, BaseURL: "http://h"}, false},
	}
	for _, c := range cases {
		got, reason := c.spec.Ready()
		if got != c.want {
			t.Fatalf("%s: want %v got %v (%s)", c.name, c.want, got, reason)
		}
		if !got && reason == "" {
			t.Fatalf("%s: 不可用时必须给出原因", c.name)
		}
	}
}

func TestLegacyCompat(t *testing.T) {
	// 完全没有新配置时，旧三段式必须能合成成 specs，保证不迁移也能跑
	cfg := &LLMConfig{
		Enable: true,
		OpenAI: &OpenAIConfig{Enable: true, APIKey: "k", Model: "gpt-4o-mini", BaseURL: "https://api.openai.com/v1"},
		Gemini: &GeminiConfig{Enable: true, APIKey: "g", Model: "gemini-3.6-flash"},
	}
	specs, sources := ResolveProviderSpecs(cfg, nil)
	if sources["openai"] != "legacy" || sources["gemini"] != "legacy" {
		t.Fatalf("旧配置应标记 legacy，实际 %+v", sources)
	}
	if ok, _ := specs["openai"].Ready(); !ok {
		t.Fatalf("openai 旧配置应可用")
	}
	if specs["ollama"] != nil {
		t.Fatalf("未启用的 ollama 不应出现在结果里")
	}
}

func TestSceneModelPrecedence(t *testing.T) {
	p := &LLMPlugin{
		Config: &LLMConfig{
			Enable:  true,
			Default: "gemini",
			Model:   "gemini-2.5-flash",
			Scenes: map[string]*SceneSelection{
				"comic.tagging": {Model: "gemini-2.5-flash-lite"},
				"comic.summary": {Model: ""},
			},
		},
		providerSpecs: map[string]*ProviderSpec{
			"gemini": {ID: "gemini", Kind: KindGemini, Enable: true, DefaultModel: "gemini-3.6-flash"},
		},
	}
	if got := p.resolveModelLocked("comic.tagging"); got != "gemini-2.5-flash-lite" {
		t.Fatalf("场景覆盖应优先，实际 %q", got)
	}
	if got := p.resolveModelLocked("comic.summary"); got != "gemini-2.5-flash" {
		t.Fatalf("空场景值应回落到服务级模型，实际 %q", got)
	}
	if got := p.resolveModelLocked(""); got != "gemini-2.5-flash" {
		t.Fatalf("无场景应使用服务级模型，实际 %q", got)
	}
	p.Config.Model = ""
	if got := p.resolveModelLocked(""); got != "gemini-3.6-flash" {
		t.Fatalf("无服务级模型应回落到 provider 默认，实际 %q", got)
	}
}
