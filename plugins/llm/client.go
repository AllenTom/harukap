package llm

import (
	"context"
	"fmt"

	"github.com/rozoomcool/go-ollama-sdk"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/genai"
)

// Message 聊天消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatOptions 聊天选项
type ChatOptions struct {
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"maxTokens,omitempty"`
}

// LLMClient LLM客户端接口
type LLMClient interface {
	Chat(messages []Message, options *ChatOptions) (string, error)
	GenerateText(ctx context.Context, prompt string) (string, error)
}

// LLMProvider LLM提供商接口
type LLMProvider interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
	GetProviderName() string
}

// OpenAIProvider OpenAI提供商
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

func NewOpenAIProvider(config *OpenAIConfig) *OpenAIProvider {
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)
	return &OpenAIProvider{
		client: client,
		model:  config.Model,
	}
}

func (p *OpenAIProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: p.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) GetProviderName() string {
	return "openai"
}

// OllamaProvider Ollama提供商
type OllamaProvider struct {
	client *ollama.OllamaClient
	model  string
}

func NewOllamaProvider(config *OllamaConfig) *OllamaProvider {
	client := ollama.NewClient(config.BaseURL)
	return &OllamaProvider{
		client: client,
		model:  config.Model,
	}
}

func (p *OllamaProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	result, err := p.client.Generate(p.model, prompt)
	if err != nil {
		return "", fmt.Errorf("Ollama API error: %w", err)
	}

	return result, nil
}

func (p *OllamaProvider) GetProviderName() string {
	return "ollama"
}

// GeminiProvider Gemini提供商
type GeminiProvider struct {
	client *genai.Client
	model  string
}

func NewGeminiProvider(ctx context.Context, config *GeminiConfig) (*GeminiProvider, error) {
	var client *genai.Client
	var err error

	// 根据配置选择使用Gemini API还是Vertex AI
	if config.Project != "" && config.Location != "" {
		// 使用Vertex AI
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			Project:  config.Project,
			Location: config.Location,
			Backend:  genai.BackendVertexAI,
		})
	} else {
		// 使用Gemini Developer API
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  config.APIKey,
			Backend: genai.BackendGeminiAPI,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiProvider{
		client: client,
		model:  config.Model,
	}, nil
}

func (p *GeminiProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	parts := []*genai.Part{
		{Text: prompt},
	}

	result, err := p.client.Models.GenerateContent(ctx, p.model, []*genai.Content{{Parts: parts}}, nil)
	if err != nil {
		return "", fmt.Errorf("Gemini API error: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func (p *GeminiProvider) GetProviderName() string {
	return "gemini"
}

func (p *GeminiProvider) Close() error {
	// Gemini client目前不需要显式关闭
	return nil
}
