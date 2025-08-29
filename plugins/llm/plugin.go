package llm

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/allentom/harukap"
	"github.com/project-xpolaris/youplustoolkit/youlog"
)

// NewPlugin 创建新的LLM插件实例
func NewPlugin() *LLMPlugin {
	return &LLMPlugin{
		providers: make(map[string]LLMProvider),
	}
}

// LLMPlugin LLM插件
type LLMPlugin struct {
	Config    *LLMConfig
	logger    *youlog.Scope
	providers map[string]LLMProvider
	engine    *harukap.HarukaAppEngine
	mutex     sync.RWMutex // 保护配置和提供商的并发访问
}

// SetConfig 设置配置
func (p *LLMPlugin) SetConfig(config LLMConfig) {
	p.Config = &config
}

// OnInit 初始化插件
func (p *LLMPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	p.logger = e.LoggerPlugin.Logger.NewScope("LLMPlugin")
	p.engine = e
	p.providers = make(map[string]LLMProvider)

	// 如果没有配置，从配置文件加载
	if p.Config == nil {
		p.logger.Info("no config, use default config source")
		p.Config = &LLMConfig{
			Enable:  e.ConfigProvider.Manager.GetBool("llm.enable"),
			Default: e.ConfigProvider.Manager.GetString("llm.default"),
		}

		// 加载OpenAI配置
		if e.ConfigProvider.Manager.GetBool("llm.openai.enable") {
			p.Config.OpenAI = &OpenAIConfig{
				Enable:  e.ConfigProvider.Manager.GetBool("llm.openai.enable"),
				APIKey:  e.ConfigProvider.Manager.GetString("llm.openai.api_key"),
				BaseURL: e.ConfigProvider.Manager.GetString("llm.openai.base_url"),
				Model:   e.ConfigProvider.Manager.GetString("llm.openai.model"),
			}
			if p.Config.OpenAI.Model == "" {
				p.Config.OpenAI.Model = "gpt-3.5-turbo" // 默认模型
			}
		}

		// 加载Ollama配置
		if e.ConfigProvider.Manager.GetBool("llm.ollama.enable") {
			p.Config.Ollama = &OllamaConfig{
				Enable:  e.ConfigProvider.Manager.GetBool("llm.ollama.enable"),
				BaseURL: e.ConfigProvider.Manager.GetString("llm.ollama.base_url"),
				Model:   e.ConfigProvider.Manager.GetString("llm.ollama.model"),
			}
			if p.Config.Ollama.BaseURL == "" {
				p.Config.Ollama.BaseURL = "http://localhost:11434" // 默认地址
			}
			if p.Config.Ollama.Model == "" {
				p.Config.Ollama.Model = "llama2" // 默认模型
			}
		}

		// 加载Gemini配置
		if e.ConfigProvider.Manager.GetBool("llm.gemini.enable") {
			p.Config.Gemini = &GeminiConfig{
				Enable:   e.ConfigProvider.Manager.GetBool("llm.gemini.enable"),
				APIKey:   e.ConfigProvider.Manager.GetString("llm.gemini.api_key"),
				Model:    e.ConfigProvider.Manager.GetString("llm.gemini.model"),
				Location: e.ConfigProvider.Manager.GetString("llm.gemini.location"),
				Project:  e.ConfigProvider.Manager.GetString("llm.gemini.project"),
			}
			if p.Config.Gemini.Model == "" {
				p.Config.Gemini.Model = "gemini-pro" // 默认模型
			}
		}

		// 设置默认提供商
		if p.Config.Default == "" {
			if p.Config.OpenAI != nil && p.Config.OpenAI.Enable {
				p.Config.Default = "openai"
			} else if p.Config.Ollama != nil && p.Config.Ollama.Enable {
				p.Config.Default = "ollama"
			} else if p.Config.Gemini != nil && p.Config.Gemini.Enable {
				p.Config.Default = "gemini"
			}
		}

		// 初始化模板配置
		if p.Config.TemplateConfig == nil {
			p.Config.TemplateConfig = &TemplateConfig{
				BusinessScenarios: make(map[string]*BusinessScenarioConfig),
			}
		}
		// 加载预定义的业务场景
		p.Config.TemplateConfig.InitializeWithPredefinedScenarios()
	}

	p.logger.WithFields(map[string]interface{}{
		"enable":  p.Config.Enable,
		"default": p.Config.Default,
		"openai":  p.Config.OpenAI != nil && p.Config.OpenAI.Enable,
		"ollama":  p.Config.Ollama != nil && p.Config.Ollama.Enable,
		"gemini":  p.Config.Gemini != nil && p.Config.Gemini.Enable,
	}).Info("LLM plugin config")

	if !p.Config.Enable {
		p.logger.Info("LLM plugin is disabled")
		return nil
	}

	// 初始化各个提供商
	ctx := context.Background()
	if err := p.initProviders(ctx); err != nil {
		return fmt.Errorf("failed to initialize LLM providers: %w", err)
	}

	p.logger.Info("LLM plugin initialized successfully")
	return nil
}

// initProviders 初始化各个LLM提供商
func (p *LLMPlugin) initProviders(ctx context.Context) error {
	// 初始化OpenAI提供商
	if p.Config.OpenAI != nil && p.Config.OpenAI.Enable {
		if p.Config.OpenAI.APIKey == "" {
			p.logger.Warn("OpenAI API key is empty, skipping OpenAI provider")
		} else {
			provider := NewOpenAIProvider(p.Config.OpenAI)
			p.providers["openai"] = provider
			p.logger.WithFields(map[string]interface{}{
				"model": p.Config.OpenAI.Model,
			}).Info("OpenAI provider initialized")
		}
	}

	// 初始化Ollama提供商
	if p.Config.Ollama != nil && p.Config.Ollama.Enable {
		provider := NewOllamaProvider(p.Config.Ollama)
		p.providers["ollama"] = provider
		p.logger.WithFields(map[string]interface{}{
			"base_url": p.Config.Ollama.BaseURL,
			"model":    p.Config.Ollama.Model,
		}).Info("Ollama provider initialized")
	}

	// 初始化Gemini提供商
	if p.Config.Gemini != nil && p.Config.Gemini.Enable {
		if p.Config.Gemini.APIKey == "" && (p.Config.Gemini.Project == "" || p.Config.Gemini.Location == "") {
			p.logger.Warn("Gemini API key or Vertex AI credentials are empty, skipping Gemini provider")
		} else {
			provider, err := NewGeminiProvider(ctx, p.Config.Gemini)
			if err != nil {
				p.logger.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Error("failed to initialize Gemini provider")
			} else {
				p.providers["gemini"] = provider
				p.logger.WithFields(map[string]interface{}{
					"model": p.Config.Gemini.Model,
				}).Info("Gemini provider initialized")
			}
		}
	}

	if len(p.providers) == 0 {
		return fmt.Errorf("no LLM providers are available")
	}

	return nil
}

// GetPluginConfig 获取插件配置
func (p *LLMPlugin) GetPluginConfig() map[string]interface{} {
	if p.Config == nil {
		return nil
	}

	config := map[string]interface{}{
		"enable":  p.Config.Enable,
		"default": p.Config.Default,
	}

	if p.Config.OpenAI != nil {
		config["openai"] = map[string]interface{}{
			"enable": p.Config.OpenAI.Enable,
			"model":  p.Config.OpenAI.Model,
		}
	}

	if p.Config.Ollama != nil {
		config["ollama"] = map[string]interface{}{
			"enable":   p.Config.Ollama.Enable,
			"base_url": p.Config.Ollama.BaseURL,
			"model":    p.Config.Ollama.Model,
		}
	}

	if p.Config.Gemini != nil {
		config["gemini"] = map[string]interface{}{
			"enable": p.Config.Gemini.Enable,
			"model":  p.Config.Gemini.Model,
		}
	}

	return config
}

// GenerateText 生成文本（使用默认提供商）
func (p *LLMPlugin) GenerateText(ctx context.Context, prompt string) (string, error) {
	p.mutex.RLock()
	defaultProvider := p.Config.Default
	p.mutex.RUnlock()
	return p.GenerateTextWithProvider(ctx, prompt, defaultProvider)
}

// GenerateTextWithProvider 使用指定提供商生成文本
func (p *LLMPlugin) GenerateTextWithProvider(ctx context.Context, prompt string, providerName string) (string, error) {
	p.mutex.RLock()
	if !p.Config.Enable {
		p.mutex.RUnlock()
		return "", fmt.Errorf("LLM plugin is disabled")
	}

	provider, exists := p.providers[providerName]
	p.mutex.RUnlock()
	if !exists {
		return "", fmt.Errorf("provider %s not found or not enabled", providerName)
	}

	response, err := provider.GenerateText(ctx, prompt)
	if err != nil {
		p.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"provider": providerName,
			"prompt":   prompt[:min(100, len(prompt))], // 记录前100个字符
		}).Error("failed to generate text")
		return "", err
	}

	p.logger.WithFields(map[string]interface{}{
		"provider":        providerName,
		"prompt_length":   len(prompt),
		"response_length": len(response),
	}).Debug("text generated successfully")

	return response, nil
}

// GetAvailableProviders 获取可用的提供商列表
func (p *LLMPlugin) GetAvailableProviders() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	providers := make([]string, 0, len(p.providers))
	for name := range p.providers {
		providers = append(providers, name)
	}
	return providers
}

// GetDefaultProvider 获取默认提供商名称
func (p *LLMPlugin) GetDefaultProvider() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.Config.Default
}

// GetClient 获取LLM客户端
func (p *LLMPlugin) GetClient() (LLMClient, error) {
	p.mutex.RLock()
	enabled := p.Config.Enable
	p.mutex.RUnlock()

	if !enabled {
		return nil, fmt.Errorf("LLM plugin is disabled")
	}
	return &llmClient{plugin: p}, nil
}

// llmClient 实现LLMClient接口的客户端
type llmClient struct {
	plugin *LLMPlugin
}

// Chat 聊天方法
func (c *llmClient) Chat(messages []Message, options *ChatOptions) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("messages cannot be empty")
	}

	// 将消息转换为单个prompt（简化实现）
	var prompt string
	for _, msg := range messages {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// 使用指定的模型提供商，如果没有指定则使用默认的
	c.plugin.mutex.RLock()
	providerName := c.plugin.Config.Default
	c.plugin.mutex.RUnlock()

	if options != nil && options.Model != "" {
		// 尝试根据模型名称确定提供商
		if provider := c.determineProvider(options.Model); provider != "" {
			providerName = provider
		}
	}

	ctx := context.Background()
	return c.plugin.GenerateTextWithProvider(ctx, prompt, providerName)
}

// GenerateText 生成文本方法
func (c *llmClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	return c.plugin.GenerateText(ctx, prompt)
}

// determineProvider 根据模型名称确定提供商
func (c *llmClient) determineProvider(model string) string {
	modelLower := strings.ToLower(model)
	// 根据模型名称推断提供商
	if strings.Contains(modelLower, "gpt") || strings.Contains(modelLower, "openai") {
		return "openai"
	}
	if strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mistral") || strings.Contains(modelLower, "gemma") {
		return "ollama"
	}
	if strings.Contains(modelLower, "gemini") {
		return "gemini"
	}
	return ""
}

// UpdateConfig 动态更新配置
func (p *LLMPlugin) UpdateConfig(newConfig *LLMConfig) error {
	return p.UpdateConfigWithPersistence(newConfig, false)
}

// UpdateConfigWithPersistence 动态更新配置，可选择是否持久化
func (p *LLMPlugin) UpdateConfigWithPersistence(newConfig *LLMConfig, persist bool) error {
	if newConfig == nil {
		return fmt.Errorf("new config cannot be nil")
	}

	// 验证新配置
	if err := p.validateConfig(newConfig); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 保存旧配置，以便在失败时回滚
	oldConfig := p.Config
	oldProviders := make(map[string]LLMProvider)
	for k, v := range p.providers {
		oldProviders[k] = v
	}

	// 更新配置
	p.Config = newConfig

	// 清理旧的提供商
	p.cleanupProviders()

	// 重新初始化提供商
	ctx := context.Background()
	if err := p.initProviders(ctx); err != nil {
		// 回滚配置
		p.Config = oldConfig
		p.providers = oldProviders
		return fmt.Errorf("failed to initialize providers with new config: %w", err)
	}

	// 如果需要持久化，保存到配置文件
	if persist {
		if err := p.saveConfigToFile(); err != nil {
			p.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Warn("failed to persist config to file, but configuration is still updated in memory")
		} else {
			p.logger.Info("configuration persisted to file successfully")
		}
	}

	p.logger.Info("LLM plugin configuration updated successfully")
	return nil
}

// validateConfig 验证配置
func (p *LLMPlugin) validateConfig(config *LLMConfig) error {
	if !config.Enable {
		return nil // 如果禁用，则无需验证提供商配置
	}

	hasValidProvider := false

	// 验证OpenAI配置
	if config.OpenAI != nil && config.OpenAI.Enable {
		if config.OpenAI.APIKey == "" {
			return fmt.Errorf("OpenAI API key is required when OpenAI is enabled")
		}
		if config.OpenAI.Model == "" {
			return fmt.Errorf("OpenAI model is required when OpenAI is enabled")
		}
		hasValidProvider = true
	}

	// 验证Ollama配置
	if config.Ollama != nil && config.Ollama.Enable {
		if config.Ollama.BaseURL == "" {
			return fmt.Errorf("Ollama base URL is required when Ollama is enabled")
		}
		if config.Ollama.Model == "" {
			return fmt.Errorf("Ollama model is required when Ollama is enabled")
		}
		hasValidProvider = true
	}

	// 验证Gemini配置
	if config.Gemini != nil && config.Gemini.Enable {
		if config.Gemini.APIKey == "" && (config.Gemini.Project == "" || config.Gemini.Location == "") {
			return fmt.Errorf("Gemini API key or Vertex AI credentials are required when Gemini is enabled")
		}
		if config.Gemini.Model == "" {
			return fmt.Errorf("Gemini model is required when Gemini is enabled")
		}
		hasValidProvider = true
	}

	if !hasValidProvider {
		return fmt.Errorf("at least one LLM provider must be enabled and properly configured")
	}

	// 验证默认提供商
	if config.Default != "" {
		validDefault := false
		if config.OpenAI != nil && config.OpenAI.Enable && config.Default == "openai" {
			validDefault = true
		}
		if config.Ollama != nil && config.Ollama.Enable && config.Default == "ollama" {
			validDefault = true
		}
		if config.Gemini != nil && config.Gemini.Enable && config.Default == "gemini" {
			validDefault = true
		}
		if !validDefault {
			return fmt.Errorf("default provider '%s' is not enabled or configured", config.Default)
		}
	}

	return nil
}

// cleanupProviders 清理现有的提供商
func (p *LLMPlugin) cleanupProviders() {
	for name, provider := range p.providers {
		if geminiProvider, ok := provider.(*GeminiProvider); ok {
			if err := geminiProvider.Close(); err != nil {
				p.logger.WithFields(map[string]interface{}{
					"error":    err.Error(),
					"provider": name,
				}).Warn("failed to close provider during cleanup")
			}
		}
	}
	p.providers = make(map[string]LLMProvider)
}

// UpdateAndSaveConfig 更新配置并持久化到配置文件
func (p *LLMPlugin) UpdateAndSaveConfig(newConfig *LLMConfig) error {
	return p.UpdateConfigWithPersistence(newConfig, true)
}

// ReloadConfig 从配置文件重新加载配置
func (p *LLMPlugin) ReloadConfig() error {
	return p.ReloadConfigWithPersistence(false)
}

// ReloadConfigWithPersistence 从配置文件重新加载配置，可选择是否在重新加载后立即保存（用于配置规范化）
func (p *LLMPlugin) ReloadConfigWithPersistence(persist bool) error {
	if p.engine == nil {
		return fmt.Errorf("engine is not initialized")
	}

	// 从配置文件重新加载配置
	newConfig := &LLMConfig{
		Enable:  p.engine.ConfigProvider.Manager.GetBool("llm.enable"),
		Default: p.engine.ConfigProvider.Manager.GetString("llm.default"),
	}

	// 加载OpenAI配置
	if p.engine.ConfigProvider.Manager.GetBool("llm.openai.enable") {
		newConfig.OpenAI = &OpenAIConfig{
			Enable:  p.engine.ConfigProvider.Manager.GetBool("llm.openai.enable"),
			APIKey:  p.engine.ConfigProvider.Manager.GetString("llm.openai.api_key"),
			BaseURL: p.engine.ConfigProvider.Manager.GetString("llm.openai.base_url"),
			Model:   p.engine.ConfigProvider.Manager.GetString("llm.openai.model"),
		}
		if newConfig.OpenAI.Model == "" {
			newConfig.OpenAI.Model = "gpt-3.5-turbo"
		}
	}

	// 加载Ollama配置
	if p.engine.ConfigProvider.Manager.GetBool("llm.ollama.enable") {
		newConfig.Ollama = &OllamaConfig{
			Enable:  p.engine.ConfigProvider.Manager.GetBool("llm.ollama.enable"),
			BaseURL: p.engine.ConfigProvider.Manager.GetString("llm.ollama.base_url"),
			Model:   p.engine.ConfigProvider.Manager.GetString("llm.ollama.model"),
		}
		if newConfig.Ollama.BaseURL == "" {
			newConfig.Ollama.BaseURL = "http://localhost:11434"
		}
		if newConfig.Ollama.Model == "" {
			newConfig.Ollama.Model = "llama2"
		}
	}

	// 加载Gemini配置
	if p.engine.ConfigProvider.Manager.GetBool("llm.gemini.enable") {
		newConfig.Gemini = &GeminiConfig{
			Enable:   p.engine.ConfigProvider.Manager.GetBool("llm.gemini.enable"),
			APIKey:   p.engine.ConfigProvider.Manager.GetString("llm.gemini.api_key"),
			Model:    p.engine.ConfigProvider.Manager.GetString("llm.gemini.model"),
			Location: p.engine.ConfigProvider.Manager.GetString("llm.gemini.location"),
			Project:  p.engine.ConfigProvider.Manager.GetString("llm.gemini.project"),
		}
		if newConfig.Gemini.Model == "" {
			newConfig.Gemini.Model = "gemini-pro"
		}
	}

	// 设置默认提供商
	if newConfig.Default == "" {
		if newConfig.OpenAI != nil && newConfig.OpenAI.Enable {
			newConfig.Default = "openai"
		} else if newConfig.Ollama != nil && newConfig.Ollama.Enable {
			newConfig.Default = "ollama"
		} else if newConfig.Gemini != nil && newConfig.Gemini.Enable {
			newConfig.Default = "gemini"
		}
	}

	return p.UpdateConfigWithPersistence(newConfig, persist)
}

// GetCurrentConfig 获取当前配置的副本
func (p *LLMPlugin) GetCurrentConfig() *LLMConfig {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.Config == nil {
		return nil
	}

	// 创建配置的深拷贝
	configCopy := &LLMConfig{
		Enable:  p.Config.Enable,
		Default: p.Config.Default,
	}

	if p.Config.OpenAI != nil {
		configCopy.OpenAI = &OpenAIConfig{
			Enable:  p.Config.OpenAI.Enable,
			APIKey:  p.Config.OpenAI.APIKey,
			BaseURL: p.Config.OpenAI.BaseURL,
			Model:   p.Config.OpenAI.Model,
		}
	}

	if p.Config.Ollama != nil {
		configCopy.Ollama = &OllamaConfig{
			Enable:  p.Config.Ollama.Enable,
			BaseURL: p.Config.Ollama.BaseURL,
			Model:   p.Config.Ollama.Model,
		}
	}

	if p.Config.Gemini != nil {
		configCopy.Gemini = &GeminiConfig{
			Enable:   p.Config.Gemini.Enable,
			APIKey:   p.Config.Gemini.APIKey,
			Model:    p.Config.Gemini.Model,
			Location: p.Config.Gemini.Location,
			Project:  p.Config.Gemini.Project,
		}
	}

	// 复制模板配置
	if p.Config.TemplateConfig != nil {
		configCopy.TemplateConfig = &TemplateConfig{
			DefaultScenario:   p.Config.TemplateConfig.DefaultScenario,
			BusinessScenarios: make(map[string]*BusinessScenarioConfig),
		}

		// 深拷贝业务场景配置
		for key, scenario := range p.Config.TemplateConfig.BusinessScenarios {
			if scenario != nil {
				customTemplates := make(map[string]string)
				for k, v := range scenario.CustomTemplates {
					customTemplates[k] = v
				}

				variables := make([]string, len(scenario.Variables))
				copy(variables, scenario.Variables)

				configCopy.TemplateConfig.BusinessScenarios[key] = &BusinessScenarioConfig{
					Name:            scenario.Name,
					Description:     scenario.Description,
					DefaultTemplate: scenario.DefaultTemplate,
					CustomTemplates: customTemplates,
					ActiveTemplate:  scenario.ActiveTemplate,
					Variables:       variables,
				}
			}
		}
	}

	return configCopy
}

// RenderTemplate 根据业务场景渲染模板
// scenarioName: 业务场景名称
// variables: 模板变量映射
// 返回渲染后的文本
func (p *LLMPlugin) RenderTemplate(scenarioName string, variables map[string]string) (string, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.Config == nil || p.Config.TemplateConfig == nil {
		return "", fmt.Errorf("template config not available")
	}

	return p.Config.TemplateConfig.RenderTemplate(scenarioName, variables)
}

// GetTemplate 根据业务场景获取当前使用的模板
// scenarioName: 业务场景名称
// 返回模板内容
func (p *LLMPlugin) GetTemplate(scenarioName string) (string, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.Config == nil || p.Config.TemplateConfig == nil {
		return "", fmt.Errorf("template config not available")
	}

	return p.Config.TemplateConfig.GetTemplate(scenarioName)
}

// GetAvailableScenarios 获取所有可用的业务场景
func (p *LLMPlugin) GetAvailableScenarios() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.Config == nil || p.Config.TemplateConfig == nil {
		return []string{}
	}

	return p.Config.TemplateConfig.GetAvailableScenarios()
}

// GetScenarioInfo 获取业务场景的详细信息
func (p *LLMPlugin) GetScenarioInfo(scenarioName string) (*BusinessScenarioConfig, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.Config == nil || p.Config.TemplateConfig == nil || p.Config.TemplateConfig.BusinessScenarios == nil {
		return nil, fmt.Errorf("template config not available")
	}

	scenario, exists := p.Config.TemplateConfig.BusinessScenarios[scenarioName]
	if !exists {
		return nil, fmt.Errorf("scenario '%s' not found", scenarioName)
	}

	return scenario, nil
}

// saveConfigToFile 保存配置到配置文件
func (p *LLMPlugin) saveConfigToFile() error {
	if p.engine == nil || p.engine.ConfigProvider == nil || p.engine.ConfigProvider.Manager == nil {
		return fmt.Errorf("configuration manager is not available")
	}

	manager := p.engine.ConfigProvider.Manager

	// 记录保存前的配置状态
	p.logger.WithFields(map[string]interface{}{
		"enable":  p.Config.Enable,
		"default": p.Config.Default,
	}).Info("Starting to save LLM config to file")

	// 保存LLM基本配置
	manager.Set("llm.enable", p.Config.Enable)
	manager.Set("llm.default", p.Config.Default)

	// 保存OpenAI配置
	if p.Config.OpenAI != nil {
		manager.Set("llm.openai.enable", p.Config.OpenAI.Enable)
		manager.Set("llm.openai.api_key", p.Config.OpenAI.APIKey)
		manager.Set("llm.openai.base_url", p.Config.OpenAI.BaseURL)
		manager.Set("llm.openai.model", p.Config.OpenAI.Model)
		p.logger.WithFields(map[string]interface{}{
			"enable": p.Config.OpenAI.Enable,
			"model":  p.Config.OpenAI.Model,
		}).Info("Saved OpenAI config")
	} else {
		// 清除OpenAI配置
		manager.Set("llm.openai.enable", false)
		manager.Set("llm.openai.api_key", "")
		manager.Set("llm.openai.base_url", "")
		manager.Set("llm.openai.model", "")
		p.logger.Info("Cleared OpenAI config")
	}

	// 保存Ollama配置
	if p.Config.Ollama != nil {
		manager.Set("llm.ollama.enable", p.Config.Ollama.Enable)
		manager.Set("llm.ollama.base_url", p.Config.Ollama.BaseURL)
		manager.Set("llm.ollama.model", p.Config.Ollama.Model)
		p.logger.WithFields(map[string]interface{}{
			"enable":   p.Config.Ollama.Enable,
			"base_url": p.Config.Ollama.BaseURL,
			"model":    p.Config.Ollama.Model,
		}).Info("Saved Ollama config")
	} else {
		// 清除Ollama配置
		manager.Set("llm.ollama.enable", false)
		manager.Set("llm.ollama.base_url", "")
		manager.Set("llm.ollama.model", "")
		p.logger.Info("Cleared Ollama config")
	}

	// 保存Gemini配置
	if p.Config.Gemini != nil {
		manager.Set("llm.gemini.enable", p.Config.Gemini.Enable)
		manager.Set("llm.gemini.api_key", p.Config.Gemini.APIKey)
		manager.Set("llm.gemini.model", p.Config.Gemini.Model)
		manager.Set("llm.gemini.location", p.Config.Gemini.Location)
		manager.Set("llm.gemini.project", p.Config.Gemini.Project)
		p.logger.WithFields(map[string]interface{}{
			"enable": p.Config.Gemini.Enable,
			"model":  p.Config.Gemini.Model,
		}).Info("Saved Gemini config")
	} else {
		// 清除Gemini配置
		manager.Set("llm.gemini.enable", false)
		manager.Set("llm.gemini.api_key", "")
		manager.Set("llm.gemini.model", "")
		manager.Set("llm.gemini.location", "")
		manager.Set("llm.gemini.project", "")
		p.logger.Info("Cleared Gemini config")
	}

	// 尝试获取配置文件路径
	var configPath string
	if configFileMethod := reflect.ValueOf(manager).MethodByName("ConfigFileUsed"); configFileMethod.IsValid() {
		if results := configFileMethod.Call([]reflect.Value{}); len(results) > 0 {
			if path, ok := results[0].Interface().(string); ok {
				configPath = path
				p.logger.WithFields(map[string]interface{}{
					"config_path": configPath,
				}).Info("Found config file path")
			}
		}
	}

	// 尝试使用反射调用写入配置的方法
	managerValue := reflect.ValueOf(manager)

	// 尝试调用 WriteConfig 方法
	writeConfigMethod := managerValue.MethodByName("WriteConfig")
	if writeConfigMethod.IsValid() {
		p.logger.Info("Attempting to save config using WriteConfig method")
		results := writeConfigMethod.Call([]reflect.Value{})
		if len(results) > 0 && !results[0].IsNil() {
			if err, ok := results[0].Interface().(error); ok {
				p.logger.WithFields(map[string]interface{}{
					"error":       err.Error(),
					"config_path": configPath,
				}).Error("WriteConfig method failed")
				return fmt.Errorf("failed to write config to file %s: %w", configPath, err)
			}
		}
		p.logger.WithFields(map[string]interface{}{
			"config_path": configPath,
		}).Info("Successfully saved config using WriteConfig method")
		return nil
	}

	// 尝试调用 SaveConfig 方法
	saveConfigMethod := managerValue.MethodByName("SaveConfig")
	if saveConfigMethod.IsValid() {
		p.logger.Info("Attempting to save config using SaveConfig method")
		results := saveConfigMethod.Call([]reflect.Value{})
		if len(results) > 0 && !results[0].IsNil() {
			if err, ok := results[0].Interface().(error); ok {
				p.logger.WithFields(map[string]interface{}{
					"error":       err.Error(),
					"config_path": configPath,
				}).Error("SaveConfig method failed")
				return fmt.Errorf("failed to save config to file %s: %w", configPath, err)
			}
		}
		p.logger.WithFields(map[string]interface{}{
			"config_path": configPath,
		}).Info("Successfully saved config using SaveConfig method")
		return nil
	}

	// 尝试调用 WriteConfigAs 方法（如果有配置文件路径）
	if configPath != "" {
		writeConfigAsMethod := managerValue.MethodByName("WriteConfigAs")
		if writeConfigAsMethod.IsValid() {
			p.logger.WithFields(map[string]interface{}{
				"config_path": configPath,
			}).Info("Attempting to save config using WriteConfigAs method")

			pathValue := reflect.ValueOf(configPath)
			results := writeConfigAsMethod.Call([]reflect.Value{pathValue})
			if len(results) > 0 && !results[0].IsNil() {
				if err, ok := results[0].Interface().(error); ok {
					p.logger.WithFields(map[string]interface{}{
						"error":       err.Error(),
						"config_path": configPath,
					}).Error("WriteConfigAs method failed")
					return fmt.Errorf("failed to write config to file %s: %w", configPath, err)
				}
			}
			p.logger.WithFields(map[string]interface{}{
				"config_path": configPath,
			}).Info("Successfully saved config using WriteConfigAs method")
			return nil
		}
	}

	// 尝试调用 SafeWriteConfig 方法
	safeWriteConfigMethod := managerValue.MethodByName("SafeWriteConfig")
	if safeWriteConfigMethod.IsValid() {
		p.logger.Info("Attempting to save config using SafeWriteConfig method")
		results := safeWriteConfigMethod.Call([]reflect.Value{})
		if len(results) > 0 && !results[0].IsNil() {
			if err, ok := results[0].Interface().(error); ok {
				// SafeWriteConfig 可能因为文件已存在而失败，这是正常的
				p.logger.WithFields(map[string]interface{}{
					"error":       err.Error(),
					"config_path": configPath,
				}).Warn("SafeWriteConfig method returned error (may be normal if file exists)")
			}
		} else {
			p.logger.WithFields(map[string]interface{}{
				"config_path": configPath,
			}).Info("Successfully saved config using SafeWriteConfig method")
			return nil
		}
	}

	// 如果没有找到合适的方法，返回警告但包含详细信息
	availableMethods := []string{}
	for i := 0; i < managerValue.NumMethod(); i++ {
		methodName := managerValue.Type().Method(i).Name
		if strings.Contains(strings.ToLower(methodName), "config") || strings.Contains(strings.ToLower(methodName), "write") || strings.Contains(strings.ToLower(methodName), "save") {
			availableMethods = append(availableMethods, methodName)
		}
	}

	p.logger.WithFields(map[string]interface{}{
		"available_methods": availableMethods,
		"config_path":       configPath,
		"manager_type":      fmt.Sprintf("%T", manager),
	}).Warn("configuration manager does not support automatic file writing - config updated in memory only")

	return fmt.Errorf("configuration manager does not support automatic file writing. Available methods: %v, Config path: %s", availableMethods, configPath)
}

// SaveConfig 保存当前配置到配置文件
func (p *LLMPlugin) SaveConfig() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.Config == nil {
		return fmt.Errorf("no configuration to save")
	}

	return p.saveConfigToFile()
}

// Close 关闭插件，清理资源
func (p *LLMPlugin) Close() error {
	for name, provider := range p.providers {
		if geminiProvider, ok := provider.(*GeminiProvider); ok {
			if err := geminiProvider.Close(); err != nil {
				p.logger.WithFields(map[string]interface{}{
					"error":    err.Error(),
					"provider": name,
				}).Error("failed to close provider")
			}
		}
	}
	return nil
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
