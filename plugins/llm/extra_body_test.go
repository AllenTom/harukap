package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureTripper 把收到的请求体存下来，替代真实 transport。
type captureTripper struct {
	body    []byte
	headers http.Header
}

func (c *captureTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		c.body, _ = io.ReadAll(req.Body)
	}
	c.headers = req.Header.Clone()
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(http.NoBody),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func newJSONRequest(payload string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)
	req.Body = io.NopCloser(strings.NewReader(payload))
	req.ContentLength = int64(len(payload))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHeaderTripperInjectsExtraBody(t *testing.T) {
	cap := &captureTripper{}
	tripper := &headerTripper{
		base:      cap,
		headers:   map[string]string{"x-opencode-session": "polaris"},
		extraBody: map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}},
	}
	req := newJSONRequest(`{"model":"longcat-2.5-free","messages":[{"role":"user","content":"hi"}],"max_tokens":512}`)
	if _, err := tripper.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(cap.body, &got); err != nil {
		t.Fatalf("请求体不是合法 JSON: %v (%s)", err, cap.body)
	}
	thinking, ok := got["thinking"].(map[string]interface{})
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("thinking 未注入: %v", got["thinking"])
	}
	// 原有字段必须原样保留
	if got["model"] != "longcat-2.5-free" || got["max_tokens"] != float64(512) {
		t.Fatalf("原有字段被破坏: %v", got)
	}
	if cap.headers.Get("x-opencode-session") != "polaris" {
		t.Fatalf("header 未注入: %v", cap.headers)
	}
	if cap.headers.Get("Content-Length") == "" {
		t.Fatalf("Content-Length 未更新")
	}
}

// 配置里的字段不能覆盖 SDK 写的字段：否则改配置就能悄悄换掉 model/messages。
func TestInjectBodyFieldsDoesNotOverrideExisting(t *testing.T) {
	req := newJSONRequest(`{"model":"real-model","max_tokens":7}`)
	err := injectBodyFields(req, map[string]interface{}{
		"model":      "hijacked",
		"max_tokens": 99999,
		"thinking":   map[string]interface{}{"type": "disabled"},
	})
	if err != nil {
		t.Fatalf("injectBodyFields: %v", err)
	}
	raw, _ := io.ReadAll(req.Body)
	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("不是合法 JSON: %v", err)
	}
	if got["model"] != "real-model" {
		t.Fatalf("model 被覆盖成 %v", got["model"])
	}
	if got["max_tokens"] != float64(7) {
		t.Fatalf("max_tokens 被覆盖成 %v", got["max_tokens"])
	}
	if _, ok := got["thinking"]; !ok {
		t.Fatalf("thinking 未注入")
	}
}

// 非 JSON 请求体（比如某些上游要表单）必须原样放行，不能被改坏。
func TestInjectBodyFieldsSkipsNonJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://example.com/x", nil)
	req.Body = io.NopCloser(strings.NewReader("a=1&b=2"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := injectBodyFields(req, map[string]interface{}{"thinking": "x"}); err != nil {
		t.Fatalf("injectBodyFields: %v", err)
	}
	raw, _ := io.ReadAll(req.Body)
	if string(raw) != "a=1&b=2" {
		t.Fatalf("非 JSON 体被改动: %q", raw)
	}
}
