package llm

import (
	"strings"
)

// 这一层把「配置是什么、为什么不可用」序列化成 HTTP 载荷。
//
// 放在 harukap 而不是各服务：四个服务（video/photo/comic/music）的设置页
// 必须口径一致，否则同一个提供商在不同模块里显示不同的 source/ready，
// 排查时无法互相对照。
//
// 兼容策略：老的 `providers`（[]string）保留，新信息放在
// `provider_details` / `scenes` / `model` 里。老前端不改也能跑。

// StatusPayload 构造 /llm/status 的响应体。
func (p *LLMPlugin) StatusPayload() map[string]any {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	ids := make([]string, 0, len(p.providers))
	for id := range p.providers {
		ids = append(ids, id)
	}
	// 稳定顺序：前端做 diff 时不会因为 map 顺序抖动而闪烁
	sortStrings(ids)

	payload := map[string]any{
		"available":        len(p.providers) > 0,
		"enable":           false,
		"default":          "",
		"model":            "",
		"providers_file":   "",
		"providers":        ids, // 兼容旧字段：仅就绪的 provider id
		"provider_details": []ResolvedProvider{},
		"scenes":           []SceneInfo{},
	}
	if p.Config == nil {
		return payload
	}
	payload["enable"] = p.Config.Enable
	payload["default"] = p.Config.Default
	payload["model"] = p.Config.Model
	payload["providers_file"] = p.Config.ProvidersFile
	payload["provider_details"] = DescribeProviders(p.providerSpecs, p.providerSources)
	payload["scenes"] = DescribeScenes(p.Config, p.providerSpecs)
	payload["guard"] = p.guardPayload()
	payload["usage"] = p.usageSnapshot()
	return payload
}

// guardPayload 暴露成本护栏的生效值，让前端能显示"这次是按什么规则在限制"。
func (p *LLMPlugin) guardPayload() map[string]any {
	return map[string]any{
		"max_output_tokens":    p.Config.EffectiveMaxOutputTokens(),
		"call_timeout_seconds": int(p.Config.EffectiveCallTimeout().Seconds()),
		"daily_call_limit":     p.Config.EffectiveDailyCallLimit(),
	}
}

// usageSnapshot 今日用量。p.Config 为 nil 时（未初始化）也要能返回，避免前端空表。
func (p *LLMPlugin) usageSnapshot() UsageSnapshot {
	if p.usage == nil {
		return UsageSnapshot{Day: today(), Remaining: -1}
	}
	limit := 0
	if p.Config != nil {
		limit = p.Config.EffectiveDailyCallLimit()
	}
	return p.usage.snapshot(limit)
}

// ConfigPayload 构造 /llm/config 的响应体。
//
// 凭据只暴露 has_key，不回传 api_key——设置页没有任何地方需要看到明文，
// 看到明文只会让浏览器历史/日志多一份泄漏面。
func (p *LLMPlugin) ConfigPayload() map[string]any {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	out := map[string]any{
		"enable":           false,
		"default":          "",
		"model":            "",
		"providers_file":   "",
		"providers":        map[string]any{},
		"scenes":           map[string]any{},
		"provider_details": []ResolvedProvider{},
		"scene_details":    []SceneInfo{},
	}
	if p.Config == nil {
		return out
	}
	cfg := p.Config
	out["enable"] = cfg.Enable
	out["default"] = cfg.Default
	out["model"] = cfg.Model
	out["providers_file"] = cfg.ProvidersFile

	// 本地声明（继承 / 覆盖），凭据一律只给 has_key
	providers := make(map[string]any, len(cfg.Providers))
	for id, ov := range cfg.Providers {
		if ov == nil {
			continue
		}
		mode := "inherit"
		if ov.IsOverride() {
			mode = "override"
		}
		providers[id] = map[string]any{
			"mode":          mode,
			"kind":          string(ov.Kind),
			"label":         ov.Label,
			"enable":        ov.Enable,
			"base_url":      ov.BaseURL,
			"project":       ov.Project,
			"location":      ov.Location,
			"default_model": ov.DefaultModel,
			"has_key":       strings.TrimSpace(ov.APIKey) != "",
			"models":        ov.Models,
		}
	}
	out["providers"] = providers

	scenes := make(map[string]any, len(cfg.Scenes))
	for id, sel := range cfg.Scenes {
		if sel == nil {
			continue
		}
		scenes[id] = map[string]any{"model": sel.Model}
	}
	out["scenes"] = scenes

	out["provider_details"] = DescribeProviders(p.providerSpecs, p.providerSources)
	out["scene_details"] = DescribeScenes(cfg, p.providerSpecs)
	out["guard"] = p.guardPayload()
	out["usage"] = p.usageSnapshot()

	// 旧三段式，保持老前端能读
	if cfg.OpenAI != nil {
		out["openai"] = map[string]any{
			"enable": cfg.OpenAI.Enable, "model": cfg.OpenAI.Model,
			"base_url": cfg.OpenAI.BaseURL, "has_key": cfg.OpenAI.APIKey != "",
		}
	}
	if cfg.Ollama != nil {
		out["ollama"] = map[string]any{
			"enable": cfg.Ollama.Enable, "model": cfg.Ollama.Model,
			"base_url": cfg.Ollama.BaseURL,
		}
	}
	if cfg.Gemini != nil {
		out["gemini"] = map[string]any{
			"enable": cfg.Gemini.Enable, "model": cfg.Gemini.Model,
			"location": cfg.Gemini.Location, "project": cfg.Gemini.Project,
			"has_key": cfg.Gemini.APIKey != "",
		}
	}
	return out
}

// UpdateRequest 是 PUT /llm/config 的请求体。
//
// 全部用指针：设置页是"局部编辑"，缺字段必须表示"别动"。
// 老实现直接 new 一个 LLMConfig 覆盖，导致任何一次保存都会把场景配置抹掉。
type UpdateRequest struct {
	Enable        *bool                        `json:"enable,omitempty"`
	Default       *string                      `json:"default,omitempty"`
	Model         *string                      `json:"model,omitempty"`
	ProvidersFile *string                      `json:"providers_file,omitempty"`
	Providers     map[string]*ProviderOverride `json:"providers,omitempty"`
	Scenes        map[string]*SceneSelection   `json:"scenes,omitempty"`

	// 成本护栏（缺省即不改）
	MaxOutputTokens    *int `json:"max_output_tokens,omitempty"`
	CallTimeoutSeconds *int `json:"call_timeout_seconds,omitempty"`
	DailyCallLimit     *int `json:"daily_call_limit,omitempty"`

	// 旧三段式（兼容老前端）
	OpenAI *OpenAIConfig `json:"openai,omitempty"`
	Ollama *OllamaConfig `json:"ollama,omitempty"`
	Gemini *GeminiConfig `json:"gemini,omitempty"`

	Persist bool `json:"persist,omitempty"`
}

// ApplyUpdate 把局部修改合并进当前配置并（可选）落盘。
func (p *LLMPlugin) ApplyUpdate(req *UpdateRequest) error {
	if req == nil {
		return nil
	}

	p.mutex.RLock()
	cur := p.Config
	var snapshot LLMConfig
	if cur != nil {
		snapshot = *cur
	}
	p.mutex.RUnlock()

	next := snapshot // 复制结构体，指针字段下面显式深拷贝

	// 场景：整体替换（未出现的场景保留）
	if req.Scenes != nil {
		scenes := make(map[string]*SceneSelection, len(next.Scenes)+len(req.Scenes))
		for k, v := range next.Scenes {
			scenes[k] = v
		}
		for k, v := range req.Scenes {
			if v == nil {
				delete(scenes, k)
				continue
			}
			scenes[k] = &SceneSelection{Model: strings.TrimSpace(v.Model)}
		}
		next.Scenes = scenes
	}

	// 本地 provider 声明：同样按 id 逐个替换，未提交的保留
	if req.Providers != nil {
		ps := make(map[string]*ProviderOverride, len(next.Providers)+len(req.Providers))
		for k, v := range next.Providers {
			ps[k] = v
		}
		for k, v := range req.Providers {
			if v == nil {
				delete(ps, k)
				continue
			}
			cp := *v
			cp.ID = k
			if cp.Mode == "" {
				cp.Mode = "inherit"
			}
			cp.Normalize()
			ps[k] = &cp
		}
		next.Providers = ps
	}

	if req.Enable != nil {
		next.Enable = *req.Enable
	}
	if req.Default != nil {
		next.Default = strings.TrimSpace(*req.Default)
	}
	if req.Model != nil {
		next.Model = strings.TrimSpace(*req.Model)
	}
	if req.ProvidersFile != nil {
		next.ProvidersFile = strings.TrimSpace(*req.ProvidersFile)
	}
	if req.MaxOutputTokens != nil {
		next.MaxOutputTokens = *req.MaxOutputTokens
	}
	if req.CallTimeoutSeconds != nil {
		next.CallTimeoutSeconds = *req.CallTimeoutSeconds
	}
	if req.DailyCallLimit != nil {
		next.DailyCallLimit = *req.DailyCallLimit
	}

	// 旧三段式：空 api_key 视为"不修改"，避免设置页不填 key 就把已有 key 清掉
	if req.OpenAI != nil {
		apiKey := req.OpenAI.APIKey
		if apiKey == "" && next.OpenAI != nil {
			apiKey = next.OpenAI.APIKey
		}
		next.OpenAI = &OpenAIConfig{
			Enable: req.OpenAI.Enable, Model: req.OpenAI.Model,
			BaseURL: req.OpenAI.BaseURL, APIKey: apiKey,
		}
	}
	if req.Ollama != nil {
		next.Ollama = &OllamaConfig{
			Enable: req.Ollama.Enable, Model: req.Ollama.Model, BaseURL: req.Ollama.BaseURL,
		}
	}
	if req.Gemini != nil {
		apiKey := req.Gemini.APIKey
		if apiKey == "" && next.Gemini != nil {
			apiKey = next.Gemini.APIKey
		}
		next.Gemini = &GeminiConfig{
			Enable: req.Gemini.Enable, Model: req.Gemini.Model, APIKey: apiKey,
			Project: req.Gemini.Project, Location: req.Gemini.Location,
		}
	}

	return p.UpdateConfigWithPersistence(&next, req.Persist)
}

// sortStrings 小工具，避免为一个排序引入额外依赖。
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
