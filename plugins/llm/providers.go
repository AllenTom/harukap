package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"google.golang.org/genai"
)

// ProviderKind 决定与上游通信使用哪种协议。
//
// 借鉴 one-api 的结论：几乎所有厂商都提供 OpenAI 兼容端点（DeepSeek、通义、
// Kimi、智谱、硅基流动、OpenRouter、vLLM、one-api/new-api 自身……），
// 所以只要有 openai-compatible 这一种，就能接住绝大多数上游；
// 只有 Gemini 的原生 SDK（尤其 Vertex 鉴权）和 Ollama 的原生 /api/generate
// 需要单独实现。新增厂商 = 新增一条配置，不需要写代码。
type ProviderKind string

const (
	// KindOpenAICompatible 任何 OpenAI 兼容端点（含自建中转）
	KindOpenAICompatible ProviderKind = "openai-compatible"
	// KindOllama Ollama 原生协议
	KindOllama ProviderKind = "ollama"
	// KindGemini Gemini Developer API 或 Vertex AI
	KindGemini ProviderKind = "gemini"
	// KindAnthropic Anthropic Messages 协议（Claude 原生，以及 MiniMax / 智谱 / Kimi
	// 提供的 /anthropic 兼容端点）
	KindAnthropic ProviderKind = "anthropic"
)

func (k ProviderKind) valid() bool {
	switch k {
	case KindOpenAICompatible, KindOllama, KindGemini, KindAnthropic:
		return true
	}
	return false
}

// ModelSpec 一个可选模型。
// Tier 用于"小白不配也能得到合理默认"：fast / balanced / quality。
type ModelSpec struct {
	ID    string `json:"id" yaml:"id"`
	Label string `json:"label,omitempty" yaml:"label,omitempty"`
	Tier  string `json:"tier,omitempty" yaml:"tier,omitempty"`
}

// ProviderSpec 一个提供商的完整描述（凭据 + 端点 + 模型）。
// 它同时是"全局凭据文件"和"服务本地覆盖"的唯一数据形态，
// 这样继承与覆盖只是"用哪一份 spec"，不涉及字段级合并。
type ProviderSpec struct {
	ID       string            `json:"id" yaml:"-"`
	Kind     ProviderKind      `json:"kind" yaml:"kind"`
	Label    string            `json:"label,omitempty" yaml:"label,omitempty"`
	Enable   bool              `json:"enable" yaml:"enable"`
	BaseURL  string            `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	APIKey   string            `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Project  string            `json:"project,omitempty" yaml:"project,omitempty"`
	Location string            `json:"location,omitempty" yaml:"location,omitempty"`
	Headers  map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Timeout  int               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	// MaxTokens 单次生成的最大输出 token。0 表示用插件级默认（见 LLMConfig.EffectiveMaxOutputTokens）。
	// 这里不设上限时模型会一路吐到上下文尽头，单次成本完全不可控。
	MaxTokens int `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
	// DefaultModel 该提供商未指定模型时使用
	DefaultModel string      `json:"default_model,omitempty" yaml:"default_model,omitempty"`
	Models       []ModelSpec `json:"models,omitempty" yaml:"models,omitempty"`
}

// Normalize 补全默认值，使"人写的配置"和"代码期望的配置"一致。
func (s *ProviderSpec) Normalize() {
	if s == nil {
		return
	}
	if s.Kind == "" {
		// 没有 kind 时按 base_url 猜：这能让老配置/手写配置少踩坑
		if strings.Contains(strings.ToLower(s.BaseURL), "11434") {
			s.Kind = KindOllama
		} else {
			s.Kind = KindOpenAICompatible
		}
	}
	switch s.Kind {
	case KindOllama:
		if s.BaseURL == "" {
			s.BaseURL = "http://localhost:11434"
		}
	case KindOpenAICompatible:
		if s.BaseURL == "" {
			s.BaseURL = "https://api.openai.com/v1"
		}
	case KindAnthropic:
		if s.BaseURL == "" {
			s.BaseURL = "https://api.anthropic.com"
		}
	}
	if s.DefaultModel == "" && len(s.Models) > 0 {
		s.DefaultModel = s.Models[0].ID
	}
}

// Ready 判断凭据是否足以发起调用，并给出**可读的原因**。
// 设计红线：不可用必须显式说明原因，不允许静默回退到别的配置。
func (s *ProviderSpec) Ready() (bool, string) {
	if s == nil {
		return false, "未配置"
	}
	if !s.Enable {
		return false, "已停用"
	}
	switch s.Kind {
	case KindOpenAICompatible:
		// 少数自建上游不要密钥，但要求显式声明（Headers 非空视作已声明鉴权方式之一）
		if strings.TrimSpace(s.APIKey) == "" && len(s.Headers) == 0 {
			return false, "缺少 API Key"
		}
		if strings.TrimSpace(s.BaseURL) == "" {
			return false, "缺少 Base URL"
		}
	case KindOllama:
		if strings.TrimSpace(s.BaseURL) == "" {
			return false, "缺少服务地址"
		}
	case KindGemini:
		vertex := strings.TrimSpace(s.Project) != "" && strings.TrimSpace(s.Location) != ""
		if !vertex && strings.TrimSpace(s.APIKey) == "" {
			return false, "缺少 API Key（或 Vertex 的项目与位置）"
		}
	case KindAnthropic:
		if strings.TrimSpace(s.APIKey) == "" && len(s.Headers) == 0 {
			return false, "缺少 API Key"
		}
	default:
		return false, fmt.Sprintf("未知类型 %q", s.Kind)
	}
	return true, ""
}

// ModelAwareProvider 支持"按次指定模型"的提供商。
//
// 借鉴 Bifrost 的做法：核心接口保持最小（LLMProvider 只有 GenerateText），
// 额外能力用可选接口 + 类型断言暴露，避免一上来就定义 60 个方法，
// 也保证老的实现不需要改动就能继续工作。
type ModelAwareProvider interface {
	LLMProvider
	GenerateTextWithModel(ctx context.Context, model, prompt string) (string, error)
}

// ModelLister 能向用户报告"这台上游实际有哪些模型"。
// 借鉴 one-api 的 GetModelList 与 Bifrost 的 ListModels：
// 模型列表属于提供商自己的能力，不该由前端硬编码。
type ModelLister interface {
	ListModels(ctx context.Context) ([]ModelSpec, error)
}

// BuildProvider 根据 spec 构造提供商实例。
func BuildProvider(ctx context.Context, spec *ProviderSpec) (LLMProvider, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil provider spec")
	}
	ok, reason := spec.Ready()
	if !ok {
		return nil, fmt.Errorf("provider %s not ready: %s", spec.ID, reason)
	}
	switch spec.Kind {
	case KindOpenAICompatible:
		return newOpenAICompatProvider(spec), nil
	case KindOllama:
		return newOllamaCompatProvider(spec), nil
	case KindGemini:
		return newGeminiCompatProvider(ctx, spec)
	case KindAnthropic:
		return newAnthropicProvider(spec), nil
	}
	return nil, fmt.Errorf("unsupported provider kind %q", spec.Kind)
}

// ---------------------------------------------------------------- OpenAI 兼容

type openAICompatProvider struct {
	id        string
	model     string
	models    []ModelSpec
	maxTokens int
	client    *openai.Client
}

func newOpenAICompatProvider(spec *ProviderSpec) *openAICompatProvider {
	cfg := openai.DefaultConfig(spec.APIKey)
	if spec.BaseURL != "" {
		cfg.BaseURL = spec.BaseURL
	}
	// go-openai 的 HTTPClient 是 HTTPDoer 接口，只能整体替换，不能改 Transport。
	timeout := 120 * time.Second
	if spec.Timeout > 0 {
		timeout = time.Duration(spec.Timeout) * time.Second
	}
	if len(spec.Headers) > 0 {
		cfg.HTTPClient = &http.Client{
			Transport: &headerTripper{base: http.DefaultTransport, headers: spec.Headers},
			Timeout:   timeout,
		}
	} else {
		cfg.HTTPClient = &http.Client{Timeout: timeout}
	}
	return &openAICompatProvider{
		id:        spec.ID,
		model:     spec.DefaultModel,
		models:    spec.Models,
		maxTokens: spec.MaxTokens,
		client:    openai.NewClientWithConfig(cfg),
	}
}

func (p *openAICompatProvider) GetProviderName() string { return p.id }

func (p *openAICompatProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	return p.GenerateTextWithModel(ctx, p.model, prompt)
}

func (p *openAICompatProvider) GenerateTextWithModel(ctx context.Context, model, prompt string) (string, error) {
	if model == "" {
		model = p.model
	}
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     model,
		Messages:  []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: prompt}},
		MaxTokens: p.maxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", p.id, err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("%s: empty response", p.id)
	}
	return resp.Choices[0].Message.Content, nil
}

// ListModels 走标准 GET {base_url}/models，兼容所有 OpenAI 兼容上游。
func (p *openAICompatProvider) ListModels(ctx context.Context) ([]ModelSpec, error) {
	resp, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s list models: %w", p.id, err)
	}
	out := make([]ModelSpec, 0, len(resp.Models))
	for _, m := range resp.Models {
		out = append(out, ModelSpec{ID: m.ID})
	}
	return out, nil
}

// ---------------------------------------------------------------- Ollama

type ollamaCompatProvider struct {
	id        string
	model     string
	baseURL   string
	maxTokens int
	httpc     *http.Client
}

func newOllamaCompatProvider(spec *ProviderSpec) *ollamaCompatProvider {
	timeout := 120 * time.Second
	if spec.Timeout > 0 {
		timeout = time.Duration(spec.Timeout) * time.Second
	}
	return &ollamaCompatProvider{
		id:        spec.ID,
		model:     spec.DefaultModel,
		baseURL:   strings.TrimRight(spec.BaseURL, "/"),
		maxTokens: spec.MaxTokens,
		httpc:     &http.Client{Timeout: timeout},
	}
}

func (p *ollamaCompatProvider) GetProviderName() string { return p.id }

func (p *ollamaCompatProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	return p.GenerateTextWithModel(ctx, p.model, prompt)
}

// GenerateTextWithModel 直接调原生 /api/generate。
//
// 原来走 go-ollama-sdk 的 client.Generate(model, prompt)：那个签名不接 ctx，
// 请求一旦发出就无法取消，批量任务里遇到慢模型只能干等。改成手写 HTTP 后
// ctx 与 num_predict（输出上限）都能生效。
func (p *ollamaCompatProvider) GenerateTextWithModel(ctx context.Context, model, prompt string) (string, error) {
	if model == "" {
		model = p.model
	}
	body := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}
	if p.maxTokens > 0 {
		body["options"] = map[string]any{"num_predict": p.maxTokens}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpc.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", p.id, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("%s: %w", p.id, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: http %d: %s", p.id, resp.StatusCode, truncate(string(data), 300))
	}
	var parsed struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("%s: 解析响应失败: %w", p.id, err)
	}
	if parsed.Error != "" {
		return "", fmt.Errorf("%s: %s", p.id, parsed.Error)
	}
	return parsed.Response, nil
}

// ListModels 走 Ollama 原生 GET /api/tags。
func (p *ollamaCompatProvider) ListModels(ctx context.Context) ([]ModelSpec, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s list models: %w", p.id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s list models: http %d", p.id, resp.StatusCode)
	}
	var body struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%s list models: %w", p.id, err)
	}
	out := make([]ModelSpec, 0, len(body.Models))
	for _, m := range body.Models {
		id := m.Name
		if id == "" {
			id = m.Model
		}
		out = append(out, ModelSpec{ID: id})
	}
	return out, nil
}

// ---------------------------------------------------------------- Gemini

type geminiCompatProvider struct {
	id        string
	model     string
	models    []ModelSpec
	maxTokens int
	client    *genai.Client
}

func newGeminiCompatProvider(ctx context.Context, spec *ProviderSpec) (*geminiCompatProvider, error) {
	var client *genai.Client
	var err error
	if strings.TrimSpace(spec.Project) != "" && strings.TrimSpace(spec.Location) != "" {
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			Project:  spec.Project,
			Location: spec.Location,
			Backend:  genai.BackendVertexAI,
		})
	} else {
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  spec.APIKey,
			Backend: genai.BackendGeminiAPI,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}
	return &geminiCompatProvider{
		id:        spec.ID,
		model:     spec.DefaultModel,
		models:    spec.Models,
		maxTokens: spec.MaxTokens,
		client:    client,
	}, nil
}

func (p *geminiCompatProvider) GetProviderName() string { return p.id }

func (p *geminiCompatProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	return p.GenerateTextWithModel(ctx, p.model, prompt)
}

func (p *geminiCompatProvider) GenerateTextWithModel(ctx context.Context, model, prompt string) (string, error) {
	if model == "" {
		model = p.model
	}
	var cfg *genai.GenerateContentConfig
	if p.maxTokens > 0 {
		cfg = &genai.GenerateContentConfig{MaxOutputTokens: int32(p.maxTokens)}
	}
	res, err := p.client.Models.GenerateContent(ctx, model,
		[]*genai.Content{{Parts: []*genai.Part{{Text: prompt}}}}, cfg)
	if err != nil {
		return "", fmt.Errorf("%s: %w", p.id, err)
	}
	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("%s: empty response", p.id)
	}
	return res.Candidates[0].Content.Parts[0].Text, nil
}

// ListModels Gemini 没有稳定的模型列举接口（Vertex 有，Developer API 没有），
// 直接返回配置里登记的候选，避免每次请求都去探活。
func (p *geminiCompatProvider) ListModels(_ context.Context) ([]ModelSpec, error) {
	if len(p.models) == 0 {
		return nil, fmt.Errorf("%s: 未配置候选模型", p.id)
	}
	return p.models, nil
}

// ---------------------------------------------------------------- 小工具

// headerTripper 给所有出站请求塞自定义头（一些中转站要求额外 header）。
type headerTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	for k, v := range t.headers {
		clone.Header.Set(k, v)
	}
	return base.RoundTrip(clone)
}
