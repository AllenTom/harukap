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
)

// Anthropic 原生协议（Messages API）。
//
// 为什么不把它塞进 openai-compatible：/v1/messages 的请求体、鉴权头
// （x-api-key + anthropic-version）、system 与 content block 的结构都和
// /chat/completions 不同，用 OpenAI 兼容端点硬转会在多模态与 tool 上出偏差。
//
// 另外 Anthropic 协议已经成了事实标准：MiniMax、智谱、Kimi 都提供
// /anthropic 兼容端点，所以这一种 kind 能接住一批国内厂商。
//
// 实现刻意用 net/http 手写，不引第三方 SDK：NAS 没有外网，
// 新增 Go 依赖会让线上无法编译。

const anthropicVersion = "2023-06-01"

type anthropicProvider struct {
	id      string
	model   string
	baseURL string
	apiKey  string
	headers map[string]string
	client  *http.Client
}

func newAnthropicProvider(spec *ProviderSpec) *anthropicProvider {
	base := strings.TrimRight(strings.TrimSpace(spec.BaseURL), "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	timeout := 120 * time.Second
	if spec.Timeout > 0 {
		timeout = time.Duration(spec.Timeout) * time.Second
	}
	return &anthropicProvider{
		id:      spec.ID,
		model:   spec.DefaultModel,
		baseURL: base,
		apiKey:  spec.APIKey,
		headers: spec.Headers,
		client:  &http.Client{Timeout: timeout},
	}
}

func (p *anthropicProvider) GetProviderName() string { return p.id }

func (p *anthropicProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	return p.GenerateTextWithModel(ctx, p.model, prompt)
}

// endpoint 处理两种 base_url 形态：
//   - https://api.anthropic.com          → /v1/messages
//   - https://api.minimax.io/anthropic/v1 → /messages
//
// 用户从文档里抄来的地址两种都见过，不能要求他背规则。
func (p *anthropicProvider) endpoint(suffix string) string {
	if strings.HasSuffix(p.baseURL, "/v1") || strings.HasSuffix(p.baseURL, "/v3") {
		return p.baseURL + suffix
	}
	return p.baseURL + "/v1" + suffix
}

func (p *anthropicProvider) do(ctx context.Context, method, url string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicVersion)
	if p.apiKey != "" {
		req.Header.Set("x-api-key", p.apiKey)
		// 部分中转只认 Bearer，两个都带上，谁认哪个都行
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p.id, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p.id, err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s: http %d: %s", p.id, resp.StatusCode, truncate(string(data), 300))
	}
	return data, nil
}

func (p *anthropicProvider) GenerateTextWithModel(ctx context.Context, model, prompt string) (string, error) {
	if model == "" {
		model = p.model
	}
	if model == "" {
		return "", fmt.Errorf("%s: 未指定模型", p.id)
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": 4096,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	}
	data, err := p.do(ctx, http.MethodPost, p.endpoint("/messages"), body)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("%s: 解析响应失败: %w", p.id, err)
	}
	var sb strings.Builder
	for _, c := range parsed.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("%s: 响应无文本内容（stop_reason=%s）", p.id, parsed.StopReason)
	}
	return sb.String(), nil
}

// ListModels 走 GET /v1/models（Anthropic 2024 起提供）。
func (p *anthropicProvider) ListModels(ctx context.Context) ([]ModelSpec, error) {
	data, err := p.do(ctx, http.MethodGet, p.endpoint("/models?limit=100"), nil)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("%s: 解析模型列表失败: %w", p.id, err)
	}
	out := make([]ModelSpec, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if m.ID == "" {
			continue
		}
		out = append(out, ModelSpec{ID: m.ID, Label: m.DisplayName})
	}
	return out, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
