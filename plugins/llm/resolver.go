package llm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// 这一层解决三件事：
//  1. 凭据公用：一份 llm-providers.yml，服务只声明"继承"或"自己填"
//  2. provider 可扩展：只要 openai-compatible 就能接，无需写代码
//  3. 场景选模型：scene.model > service.model > provider.default_model > 内置档位

// ProviderOverride 服务本地对某个提供商的声明。
// Mode=inherit（默认）表示用全局那份；Mode=override 表示整段使用本地字段。
//
// 设计红线：**不做字段级合并**。"继承 base_url 但自己填 key" 会造出
// 看起来配了、实际连不通的配置，这类谜题比多填几个字段更贵。
type ProviderOverride struct {
	Mode         string `json:"mode" yaml:"mode"`
	ProviderSpec `yaml:",inline"`
}

// IsOverride 判断是否使用了本地自定义凭据。
func (o *ProviderOverride) IsOverride() bool {
	return o != nil && strings.EqualFold(strings.TrimSpace(o.Mode), "override")
}

// SceneSelection 单个业务场景的模型覆盖。
// 只允许覆盖 model：换提供商意味着换凭据/计费，属于服务级决策。
type SceneSelection struct {
	Model string `json:"model,omitempty" yaml:"model,omitempty"`
}

// ProvidersFileContent 全局凭据文件的结构。
type ProvidersFileContent struct {
	Version int `yaml:"version" json:"version"`
	// Providers 的 key 即 provider id（deepseek / gemini / ollama / newapi…）
	Providers map[string]*ProviderSpec `yaml:"providers" json:"providers"`
}

// LoadProvidersFile 读取全局凭据文件。文件不存在时返回 nil 且不报错，
// 这样"没有这条路"能干净地退化回旧行为，而不是让服务起不来。
func LoadProvidersFile(path string) (*ProvidersFileContent, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取 providers 文件 %s: %w", path, err)
	}
	var content ProvidersFileContent
	if err := yaml.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("解析 providers 文件 %s: %w", path, err)
	}
	for id, spec := range content.Providers {
		if spec == nil {
			continue
		}
		spec.ID = id
		spec.Normalize()
	}
	return &content, nil
}

// SaveProvidersFile 原子写入共享凭据文件。
//
// 三个约束：0600（文件里是明文 Key）、先备份（用户改错了能回退）、
// rename 落盘（避免服务恰好在这时 reload，读到半个文件）。
func SaveProvidersFile(path string, content *ProvidersFileContent) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("providers 文件路径为空")
	}
	if content == nil {
		content = &ProvidersFileContent{Version: 1}
	}
	if content.Version == 0 {
		content.Version = 1
	}
	if content.Providers == nil {
		content.Providers = map[string]*ProviderSpec{}
	}
	for id, spec := range content.Providers {
		if spec == nil {
			delete(content.Providers, id)
			continue
		}
		spec.ID = id
		spec.Normalize()
	}

	data, err := yaml.Marshal(content)
	if err != nil {
		return fmt.Errorf("序列化 providers 文件: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建目录 %s: %w", dir, err)
	}

	if _, statErr := os.Stat(path); statErr == nil {
		backup := fmt.Sprintf("%s.bak.%s", path, time.Now().Format("20060102-150405"))
		if err := copyFile(path, backup); err != nil {
			return fmt.Errorf("备份 %s: %w", path, err)
		}
	}

	tmp, err := os.CreateTemp(dir, ".llm-providers-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("设置文件权限: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("写入临时文件: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("刷盘: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("替换 %s: %w", path, err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// ResolveProviderSpecs 合并"全局凭据"与"服务本地覆盖"，得到最终生效的提供商集合。
//
// 返回 specs 与 source（global / local）。无法解析或未配置的 provider 也会
// 出现在结果里（Ready=false + 原因），这样 UI 才能显示"音乐选的 openai 没配"，
// 而不是静默失败。
func ResolveProviderSpecs(cfg *LLMConfig, shared *ProvidersFileContent) (map[string]*ProviderSpec, map[string]string) {
	specs := make(map[string]*ProviderSpec)
	sources := make(map[string]string)

	if shared != nil {
		for id, spec := range shared.Providers {
			if spec == nil {
				continue
			}
			specs[id] = spec
			sources[id] = "global"
		}
	}

	// 本地覆盖
	for id, ov := range cfg.Providers {
		if ov == nil {
			continue
		}
		ov.ID = id
		if ov.Kind == "" {
			ov.Kind = specs[id].kindOr(KindOpenAICompatible)
		}
		ov.Normalize()
		if ov.IsOverride() {
			specs[id] = &ov.ProviderSpec
			sources[id] = "local"
		} else {
			// 继承：全局没有这个 provider 时，明确标记为未配置
			if _, ok := specs[id]; !ok {
				// 保留本地声明的模型等信息，但凭据来源标记为 global（缺失）
				empty := &ProviderSpec{
					ID:           id,
					Kind:         ov.Kind,
					Label:        ov.Label,
					DefaultModel: ov.DefaultModel,
					Models:       ov.Models,
				}
				empty.Enable = true // 让 Ready() 的失败原因指向"全局未配置"
				specs[id] = empty
				sources[id] = "missing"
			}
		}
	}

	// 兼容期：没有任何 providers 配置时，从旧的 openai/ollama/gemini 三段合成
	if len(cfg.Providers) == 0 && (shared == nil || len(shared.Providers) == 0) {
		for id, spec := range legacySpecs(cfg) {
			specs[id] = spec
			sources[id] = "legacy"
		}
	}

	return specs, sources
}

func (s *ProviderSpec) kindOr(fallback ProviderKind) ProviderKind {
	if s != nil && s.Kind != "" {
		return s.Kind
	}
	return fallback
}

// legacySpecs 把旧的三段式配置翻译成统一 spec，保证不改配置也能跑。
func legacySpecs(cfg *LLMConfig) map[string]*ProviderSpec {
	out := make(map[string]*ProviderSpec)
	if cfg.OpenAI != nil {
		out["openai"] = &ProviderSpec{
			ID: "openai", Kind: KindOpenAICompatible, Label: "OpenAI",
			Enable: cfg.OpenAI.Enable, BaseURL: cfg.OpenAI.BaseURL,
			APIKey: cfg.OpenAI.APIKey, DefaultModel: cfg.OpenAI.Model,
		}
	}
	if cfg.Ollama != nil {
		out["ollama"] = &ProviderSpec{
			ID: "ollama", Kind: KindOllama, Label: "Ollama",
			Enable: cfg.Ollama.Enable, BaseURL: cfg.Ollama.BaseURL,
			DefaultModel: cfg.Ollama.Model,
		}
	}
	if cfg.Gemini != nil {
		out["gemini"] = &ProviderSpec{
			ID: "gemini", Kind: KindGemini, Label: "Gemini",
			Enable: cfg.Gemini.Enable, APIKey: cfg.Gemini.APIKey,
			Project: cfg.Gemini.Project, Location: cfg.Gemini.Location,
			DefaultModel: cfg.Gemini.Model,
		}
	}
	for _, s := range out {
		s.Normalize()
	}
	return out
}

// ResolvedProvider 暴露给管理接口的提供商现状。
type ResolvedProvider struct {
	ID           string       `json:"id"`
	Kind         ProviderKind `json:"kind"`
	Label        string       `json:"label,omitempty"`
	Source       string       `json:"source"` // global | local | legacy | missing
	Ready        bool         `json:"ready"`
	Reason       string       `json:"reason,omitempty"`
	DefaultModel string       `json:"default_model,omitempty"`
	Models       []ModelSpec  `json:"models,omitempty"`
	HasKey       bool         `json:"has_key"`
}

// DescribeProviders 生成稳定的（按 id 排序的）现状列表。
func DescribeProviders(specs map[string]*ProviderSpec, sources map[string]string) []ResolvedProvider {
	out := make([]ResolvedProvider, 0, len(specs))
	for id, spec := range specs {
		ready, reason := spec.Ready()
		if sources[id] == "missing" {
			ready, reason = false, "全局未配置该提供商"
		}
		out = append(out, ResolvedProvider{
			ID: id, Kind: spec.Kind, Label: spec.Label,
			Source: sources[id], Ready: ready, Reason: reason,
			DefaultModel: spec.DefaultModel, Models: spec.Models,
			HasKey: strings.TrimSpace(spec.APIKey) != "",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ---------------------------------------------------------------- 配置加载

// LoadConfig 从 viper 读取完整配置。
// OnInit 与 ReloadConfig 共用这一处，避免两边字段漏读（历史上就是这样漏掉
// template_config 的，导致场景配置重启即丢）。
func LoadConfig(mgr *viper.Viper) *LLMConfig {
	cfg := &LLMConfig{
		Enable:        mgr.GetBool("llm.enable"),
		Default:       mgr.GetString("llm.default"),
		Model:         mgr.GetString("llm.model"),
		ProvidersFile: mgr.GetString("llm.providers_file"),
	}
	if cfg.Default == "" {
		cfg.Default = mgr.GetString("llm.provider")
	}

	if mgr.GetBool("llm.openai.enable") {
		cfg.OpenAI = &OpenAIConfig{
			Enable:  true,
			APIKey:  mgr.GetString("llm.openai.api_key"),
			BaseURL: mgr.GetString("llm.openai.base_url"),
			Model:   mgr.GetString("llm.openai.model"),
		}
		if cfg.OpenAI.Model == "" {
			cfg.OpenAI.Model = "gpt-4o-mini"
		}
	}
	if mgr.GetBool("llm.ollama.enable") {
		cfg.Ollama = &OllamaConfig{
			Enable:  true,
			BaseURL: mgr.GetString("llm.ollama.base_url"),
			Model:   mgr.GetString("llm.ollama.model"),
		}
		if cfg.Ollama.BaseURL == "" {
			cfg.Ollama.BaseURL = "http://localhost:11434"
		}
		if cfg.Ollama.Model == "" {
			cfg.Ollama.Model = "llama2"
		}
	}
	if mgr.GetBool("llm.gemini.enable") {
		cfg.Gemini = &GeminiConfig{
			Enable:   true,
			APIKey:   mgr.GetString("llm.gemini.api_key"),
			Model:    mgr.GetString("llm.gemini.model"),
			Location: mgr.GetString("llm.gemini.location"),
			Project:  mgr.GetString("llm.gemini.project"),
		}
		if cfg.Gemini.Model == "" {
			cfg.Gemini.Model = "gemini-pro"
		}
	}

	// provider 列表（服务本地声明）
	if raw := mgr.GetStringMap("llm.providers"); len(raw) > 0 {
		cfg.Providers = make(map[string]*ProviderOverride, len(raw))
		for id, v := range raw {
			ov := &ProviderOverride{}
			if m, ok := v.(map[string]any); ok {
				ov.Mode = asString(m["mode"])
				ov.Kind = ProviderKind(asString(m["kind"]))
				ov.Label = asString(m["label"])
				ov.Enable = asBool(m["enable"], true)
				ov.BaseURL = asString(m["base_url"])
				ov.APIKey = asString(m["api_key"])
				ov.Project = asString(m["project"])
				ov.Location = asString(m["location"])
				ov.DefaultModel = asString(m["default_model"])
				ov.Timeout = asInt(m["timeout"])
				ov.Models = parseModelSpecs(m["models"])
				if h, ok := m["headers"].(map[string]any); ok {
					ov.Headers = make(map[string]string, len(h))
					for k, hv := range h {
						ov.Headers[k] = asString(hv)
					}
				}
				if b, ok := m["extra_body"].(map[string]any); ok {
					ov.ExtraBody = b
				}
			}
			cfg.Providers[id] = ov
		}
	}

	// 场景级模型覆盖
	if raw := mgr.GetStringMap("llm.scenes"); len(raw) > 0 {
		cfg.Scenes = make(map[string]*SceneSelection, len(raw))
		for name, v := range raw {
			sel := &SceneSelection{}
			if m, ok := v.(map[string]any); ok {
				sel.Model = asString(m["model"])
			}
			cfg.Scenes[name] = sel
		}
	}

	// 默认提供商：没显式写就按"第一个就绪的"推导（保持旧行为）
	if cfg.Default == "" {
		if cfg.OpenAI != nil && cfg.OpenAI.Enable {
			cfg.Default = "openai"
		} else if cfg.Ollama != nil && cfg.Ollama.Enable {
			cfg.Default = "ollama"
		} else if cfg.Gemini != nil && cfg.Gemini.Enable {
			cfg.Default = "gemini"
		}
	}

	// 模板配置（含场景）也必须从这里加载，否则场景模型重启即丢
	if cfg.TemplateConfig == nil {
		cfg.TemplateConfig = &TemplateConfig{BusinessScenarios: make(map[string]*BusinessScenarioConfig)}
	}
	cfg.TemplateConfig.InitializeWithPredefinedScenarios()

	return cfg
}

func parseModelSpecs(v any) []ModelSpec {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]ModelSpec, 0, len(list))
	for _, item := range list {
		switch t := item.(type) {
		case string:
			if t != "" {
				out = append(out, ModelSpec{ID: t})
			}
		case map[string]any:
			id := asString(t["id"])
			if id == "" {
				id = asString(t["name"])
			}
			if id == "" {
				continue
			}
			out = append(out, ModelSpec{ID: id, Label: asString(t["label"]), Tier: asString(t["tier"])})
		}
	}
	return out
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func asBool(v any, def bool) bool {
	if v == nil {
		return def
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

func asInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	}
	return 0
}

// writeSelectionKeys 只写"选择项"，不写继承来的凭据。
// 本地 override 的凭据属于用户显式输入，必须落盘；inherit 的则清空，
// 防止配置文件里藏着一份已经过期的明文 Key。
func (p *LLMPlugin) writeSelectionKeys(manager *viper.Viper) {
	for id, ov := range p.Config.Providers {
		if ov == nil {
			continue
		}
		prefix := "llm.providers." + id
		if ov.IsOverride() {
			manager.Set(prefix+".mode", "override")
			manager.Set(prefix+".kind", string(ov.Kind))
			manager.Set(prefix+".label", ov.Label)
			manager.Set(prefix+".enable", ov.Enable)
			manager.Set(prefix+".base_url", ov.BaseURL)
			manager.Set(prefix+".api_key", ov.APIKey)
			manager.Set(prefix+".project", ov.Project)
			manager.Set(prefix+".location", ov.Location)
			manager.Set(prefix+".default_model", ov.DefaultModel)
		} else {
			manager.Set(prefix+".mode", "inherit")
			manager.Set(prefix+".api_key", "")
			manager.Set(prefix+".base_url", "")
		}
	}
	for name, sel := range p.Config.Scenes {
		if sel == nil {
			continue
		}
		manager.Set("llm.scenes."+name+".model", sel.Model)
	}
}
