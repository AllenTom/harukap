package llm

import (
	"fmt"
	"strings"
)

// LLMConfig LLM插件配置
type LLMConfig struct {
	Enable         bool            `json:"enable"`
	OpenAI         *OpenAIConfig   `json:"openai,omitempty"`
	Ollama         *OllamaConfig   `json:"ollama,omitempty"`
	Gemini         *GeminiConfig   `json:"gemini,omitempty"`
	Default        string          `json:"default"`                   // 默认使用的LLM提供商: "openai", "ollama", "gemini"
	TemplateConfig *TemplateConfig `json:"template_config,omitempty"` // 模板配置
}

// TemplateConfig 模板配置
type TemplateConfig struct {
	BusinessScenarios map[string]*BusinessScenarioConfig `json:"business_scenarios"` // 业务场景配置
	DefaultScenario   string                             `json:"default_scenario"`   // 默认业务场景
}

// BusinessScenarioConfig 业务场景配置
type BusinessScenarioConfig struct {
	Name            string            `json:"name"`             // 业务场景名称
	Description     string            `json:"description"`      // 业务场景描述
	DefaultTemplate string            `json:"default_template"` // 默认模板内容
	CustomTemplates map[string]string `json:"custom_templates"` // 自定义模板集合
	ActiveTemplate  string            `json:"active_template"`  // 当前使用的模板key，空表示使用默认模板
	Variables       []string          `json:"variables"`        // 该业务场景支持的变量列表
}

// OpenAIConfig OpenAI配置
type OpenAIConfig struct {
	Enable  bool   `json:"enable"`
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"` // 可选，用于支持OpenAI兼容的API
	Model   string `json:"model"`              // 默认模型，如 "gpt-3.5-turbo"
}

// OllamaConfig Ollama配置
type OllamaConfig struct {
	Enable  bool   `json:"enable"`
	BaseURL string `json:"base_url"` // Ollama服务地址，如 "http://localhost:11434"
	Model   string `json:"model"`    // 默认模型，如 "llama2"
}

// GeminiConfig Gemini配置
type GeminiConfig struct {
	Enable   bool   `json:"enable"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`    // 默认模型，如 "gemini-pro"
	Location string `json:"location"` // 用于Vertex AI，可选
	Project  string `json:"project"`  // 用于Vertex AI，可选
}

// GetTemplate 获取指定业务场景的当前模板
func (tc *TemplateConfig) GetTemplate(scenario string) (string, error) {
	if tc == nil || tc.BusinessScenarios == nil {
		return "", fmt.Errorf("template config not initialized")
	}

	scenarioConfig, exists := tc.BusinessScenarios[scenario]
	if !exists {
		return "", fmt.Errorf("business scenario '%s' not found", scenario)
	}

	// 如果设置了ActiveTemplate，使用自定义模板
	if scenarioConfig.ActiveTemplate != "" {
		if template, exists := scenarioConfig.CustomTemplates[scenarioConfig.ActiveTemplate]; exists {
			return template, nil
		}
	}

	// 否则使用默认模板
	return scenarioConfig.DefaultTemplate, nil
}

// RenderTemplate 渲染模板，将{{变量名}}替换为实际值
func (tc *TemplateConfig) RenderTemplate(scenario string, variables map[string]string) (string, error) {
	template, err := tc.GetTemplate(scenario)
	if err != nil {
		return "", err
	}

	// 替换模板中的变量
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}

// GetAvailableScenarios 获取所有可用的业务场景
func (tc *TemplateConfig) GetAvailableScenarios() []string {
	if tc == nil || tc.BusinessScenarios == nil {
		return []string{}
	}

	scenarios := make([]string, 0, len(tc.BusinessScenarios))
	for key := range tc.BusinessScenarios {
		scenarios = append(scenarios, key)
	}

	return scenarios
}

// GetPredefinedScenarios 获取预定义的业务场景配置
func GetPredefinedScenarios() map[string]*BusinessScenarioConfig {
	return map[string]*BusinessScenarioConfig{
		"tag_analysis": {
			Name:        "漫画标签分析",
			Description: "分析漫画文件名或描述文本，自动提取相关的标签信息",
			DefaultTemplate: `你是一个专业的漫画标签分析助手。请分析以下文本，提取出漫画相关的标签信息。

文本内容："{{content}}"

请从文本中提取以下类型的标签：
- artist: 画师/作者名称
- series: 系列/作品名称  
- name: 漫画标题/名称
- theme: 主题/题材标签
- translator: 翻译者
- type: 漫画类型(如CG、同人志等)
- lang: 语言
- magazine: 杂志名称
- societies: 社团名称

请以JSON格式返回结果，格式如下：
{
  "tags": [
    {"name": "标签名称", "type": "标签类型"},
    {"name": "标签名称", "type": "标签类型"}
  ]
}`,
			CustomTemplates: make(map[string]string),
			ActiveTemplate:  "",
			Variables:       []string{"content", "text"},
		},
		"content_summary": {
			Name:        "内容摘要生成",
			Description: "为漫画内容生成简洁的摘要描述",
			DefaultTemplate: `请为以下漫画生成一个简洁的内容摘要：

标题：{{title}}
类型：{{genre}}
内容：{{content}}

要求：
- 摘要长度控制在100字以内
- 突出主要情节和特色
- 避免剧透
- 语言简洁明了`,
			CustomTemplates: make(map[string]string),
			ActiveTemplate:  "",
			Variables:       []string{"title", "genre", "content"},
		},
		"translation_check": {
			Name:        "翻译质量检查",
			Description: "检查漫画翻译的准确性和流畅性",
			DefaultTemplate: `请检查以下翻译的质量：

原文：{{original}}
译文：{{translation}}
上下文：{{context}}

请评估：
1. 翻译准确性（是否正确传达原意）
2. 语言流畅性（是否自然流畅）
3. 文化适应性（是否适合目标语言文化）
4. 改进建议（如有需要）

请提供详细的评估结果和建议。`,
			CustomTemplates: make(map[string]string),
			ActiveTemplate:  "",
			Variables:       []string{"original", "translation", "context"},
		},
		"title_generation": {
			Name:        "标题生成",
			Description: "根据漫画内容特征生成合适的标题",
			DefaultTemplate: `根据以下漫画信息，请生成一个吸引人的标题：

作者：{{artist}}
类型：{{genre}}
主题：{{theme}}
简介：{{summary}}

要求：
- 标题要吸引读者注意
- 体现作品的主要特色
- 长度适中（5-15个字符）
- 符合漫画风格

请提供3-5个备选标题。`,
			CustomTemplates: make(map[string]string),
			ActiveTemplate:  "",
			Variables:       []string{"artist", "genre", "theme", "summary"},
		},
	}
}

// InitializeWithPredefinedScenarios 使用预定义场景初始化模板配置
func (tc *TemplateConfig) InitializeWithPredefinedScenarios() {
	if tc.BusinessScenarios == nil {
		tc.BusinessScenarios = make(map[string]*BusinessScenarioConfig)
	}

	predefined := GetPredefinedScenarios()
	for key, scenario := range predefined {
		// 只在场景不存在时才添加，避免覆盖用户的自定义配置
		if _, exists := tc.BusinessScenarios[key]; !exists {
			tc.BusinessScenarios[key] = scenario
		}
	}

	// 如果没有设置默认场景，设置为tag_analysis
	if tc.DefaultScenario == "" {
		tc.DefaultScenario = "tag_analysis"
	}
}
