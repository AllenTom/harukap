package llm

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/allentom/harukap"
	"github.com/allentom/harukap/plugins/youlog"
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
	// providerSpecs 是"解析后"的提供商集合（全局凭据 + 本地覆盖 + 旧配置合成）
	providerSpecs   map[string]*ProviderSpec
	providerSources map[string]string
	engine          *harukap.HarukaAppEngine
	mutex           sync.RWMutex // 保护配置和提供商的并发访问
}

// SetConfig 设置配置
func (p *LLMPlugin) SetConfig(config LLMConfig) {
	p.Config = &config
}

// OnInit 初始化插件
func (p *LLMPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	p.logger = e.LoggerPlugin.NewScope("LLMPlugin")
	p.engine = e
	p.providers = make(map[string]LLMProvider)

	// 如果没有配置，从配置文件加载
	if p.Config == nil {
		p.logger.Info("no config, use default config source")
		p.Config = LoadConfig(e.ConfigProvider.Manager)
	}

	p.logger.WithFields(map[string]interface{}{
		"enable":  p.Config.Enable,
		"default": p.Config.Default,
		"openai":  p.Config.OpenAI != nil && p.Config.OpenAI.Enable,
		"ollama":  p.Config.Ollama != nil && p.Config.Ollama.Enable,
		"gemini":  p.Config.Gemini != nil && p.Config.Gemini.Enable,
	}).Info("LLM plugin config")

	// 即使总开关关闭也要走一遍：需要把 provider 现状解析出来，
	// 管理界面才能显示"配了什么、为什么不可用"，而不是一片空白。
	ctx := context.Background()
	if err := p.initProviders(ctx); err != nil {
		if p.Config.Enable {
			return fmt.Errorf("failed to initialize LLM providers: %w", err)
		}
		p.logger.WithFields(map[string]interface{}{"error": err.Error()}).
			Warn("provider resolution failed while plugin disabled")
	}

	p.logger.Info("LLM plugin initialized successfully")
	return nil
}

// initProviders 按解析后的 provider specs 重建提供商实例。
//
// 解析来源优先级：全局凭据文件 → 服务本地 override → 旧的 openai/ollama/gemini 三段。
// 这一步同时把 p.providerSpecs 填好，供 /status 与管理接口使用。
func (p *LLMPlugin) initProviders(ctx context.Context) error {
	p.providers = make(map[string]LLMProvider)

	shared, err := LoadProvidersFile(p.Config.ProvidersFile)
	if err != nil {
		// 文件存在但解析失败：必须显式失败，不能静默当成"没配"，
		// 否则用户会以为凭据生效了，实际在用旧的残留配置。
		return err
	}
	specs, sources := ResolveProviderSpecs(p.Config, shared)
	p.providerSpecs = specs
	p.providerSources = sources

	if !p.Config.Enable {
		p.logger.Info("LLM plugin is disabled")
		return nil
	}

	ids := make([]string, 0, len(specs))
	for id := range specs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		spec := specs[id]
		ready, reason := spec.Ready()
		if sources[id] == "missing" {
			ready, reason = false, "全局未配置该提供商"
		}
		if !ready {
			p.logger.WithFields(map[string]interface{}{
				"provider": id, "reason": reason,
			}).Info("provider skipped")
			continue
		}
		provider, err := BuildProvider(ctx, spec)
		if err != nil {
			p.logger.WithFields(map[string]interface{}{
				"provider": id, "error": err.Error(),
			}).Error("failed to initialize provider")
			continue
		}
		p.providers[id] = provider
		p.logger.WithFields(map[string]interface{}{
			"provider": id, "kind": spec.Kind, "model": spec.DefaultModel,
			"source": sources[id],
		}).Info("provider initialized")
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

// GenerateText 生成文本（使用默认提供商与服务级默认模型）
func (p *LLMPlugin) GenerateText(ctx context.Context, prompt string) (string, error) {
	return p.GenerateTextForScene(ctx, "", prompt)
}

// GenerateTextForScene 按业务场景生成文本。
//
// 模型解析顺序：scene.model → service.model → provider.default_model。
// 场景为空等价于"用服务级默认"，所以老调用点不改也能工作。
func (p *LLMPlugin) GenerateTextForScene(ctx context.Context, scene, prompt string) (string, error) {
	p.mutex.RLock()
	if !p.Config.Enable {
		p.mutex.RUnlock()
		return "", fmt.Errorf("LLM plugin is disabled")
	}
	providerName := p.Config.Default
	model := p.resolveModelLocked(scene)
	p.mutex.RUnlock()

	if providerName == "" {
		return "", fmt.Errorf("no default LLM provider configured")
	}
	return p.generateWith(ctx, providerName, model, prompt)
}

// resolveModelLocked 计算某个场景实际使用的模型。调用方需持有读锁。
func (p *LLMPlugin) resolveModelLocked(scene string) string {
	model, _ := resolveSceneModel(p.Config, p.providerSpecs, scene)
	return model
}

// generateWith 用指定提供商与模型发起请求（模型为空则用该提供商默认）。
func (p *LLMPlugin) generateWith(ctx context.Context, providerName, model, prompt string) (string, error) {
	p.mutex.RLock()
	provider, exists := p.providers[providerName]
	p.mutex.RUnlock()
	if !exists {
		return "", fmt.Errorf("provider %s not found or not enabled", providerName)
	}

	var (
		response string
		err      error
	)
	if model != "" {
		if aware, ok := provider.(ModelAwareProvider); ok {
			response, err = aware.GenerateTextWithModel(ctx, model, prompt)
		} else {
			response, err = provider.GenerateText(ctx, prompt)
		}
	} else {
		response, err = provider.GenerateText(ctx, prompt)
	}

	if err != nil {
		p.logger.WithFields(map[string]interface{}{
			"error": err.Error(), "provider": providerName, "model": model,
			"prompt": prompt[:min(100, len(prompt))],
		}).Error("failed to generate text")
		return "", err
	}

	p.logger.WithFields(map[string]interface{}{
		"provider": providerName, "model": model,
		"prompt_length": len(prompt), "response_length": len(response),
	}).Debug("text generated successfully")
	return response, nil
}

// GetResolvedProviders 返回提供商现状（含来源与不可用原因），供管理接口展示。
func (p *LLMPlugin) GetResolvedProviders() []ResolvedProvider {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return DescribeProviders(p.providerSpecs, p.providerSources)
}

// GetResolvedScenes 返回场景 → 模型现状（含来源），供设置页展示。
func (p *LLMPlugin) GetResolvedScenes() []SceneInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return DescribeScenes(p.Config, p.providerSpecs)
}

// ResolveSceneModel 计算指定场景最终使用的模型与来源。
func (p *LLMPlugin) ResolveSceneModel(scene string) (string, string) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return resolveSceneModel(p.Config, p.providerSpecs, scene)
}

// ListProviderModels 读取某个提供商实际可用的模型；先问上游，失败时退回配置的候选。
func (p *LLMPlugin) ListProviderModels(ctx context.Context, providerID string) ([]ModelSpec, string, error) {
	p.mutex.RLock()
	provider, ok := p.providers[providerID]
	spec := p.providerSpecs[providerID]
	p.mutex.RUnlock()

	if lister, ok2 := provider.(ModelLister); ok2 {
		models, err := lister.ListModels(ctx)
		if err == nil && len(models) > 0 {
			return models, "upstream", nil
		}
		if spec != nil && len(spec.Models) > 0 {
			return spec.Models, "config", nil
		}
		if err != nil {
			return nil, "", err
		}
	}
	if spec != nil && len(spec.Models) > 0 {
		return spec.Models, "config", nil
	}
	if !ok {
		return nil, "", fmt.Errorf("provider %s 不可用（未初始化）", providerID)
	}
	return nil, "", fmt.Errorf("provider %s 不支持模型列举", providerID)
}

// GenerateTextWithProvider 使用指定提供商生成文本（模型用该提供商默认）
func (p *LLMPlugin) GenerateTextWithProvider(ctx context.Context, prompt string, providerName string) (string, error) {
	p.mutex.RLock()
	enabled := p.Config.Enable
	p.mutex.RUnlock()
	if !enabled {
		return "", fmt.Errorf("LLM plugin is disabled")
	}
	return p.generateWith(ctx, providerName, "", prompt)
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

// GenerateTextForScene 按场景生成文本方法
func (c *llmClient) GenerateTextForScene(ctx context.Context, scene, prompt string) (string, error) {
	return c.plugin.GenerateTextForScene(ctx, scene, prompt)
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

	// 新形态：凭据来自共享文件或本地 provider 声明。校验的是"解析后至少有一个可用"，
	// 而不是"三段式里有一段开着"——后者在共享凭据模式下恒不成立。
	if config.ProvidersFile != "" || len(config.Providers) > 0 {
		shared, err := LoadProvidersFile(config.ProvidersFile)
		if err != nil {
			return err
		}
		specs, sources := ResolveProviderSpecs(config, shared)
		readyCount := 0
		for id, spec := range specs {
			if sources[id] == "missing" {
				continue
			}
			if ok, _ := spec.Ready(); ok {
				readyCount++
			}
		}
		if readyCount == 0 {
			return fmt.Errorf("no usable LLM provider: 检查全局凭据文件或本地 provider 配置")
		}
		if config.Default != "" {
			if _, ok := specs[config.Default]; !ok {
				return fmt.Errorf("default provider '%s' is not defined", config.Default)
			}
		}
		return nil
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

	// 从配置文件重新加载配置（与 OnInit 共用同一处加载逻辑）
	newConfig := LoadConfig(p.engine.ConfigProvider.Manager)

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

	// 共享凭据模式下，本地文件只保存"用谁、用哪个模型"，绝不把继承来的凭据写回去。
	// 这是"凭据只有一份"的硬性保证——一旦写回，就会退化成四份并用。
	sharedMode := strings.TrimSpace(p.Config.ProvidersFile) != ""

	// 记录保存前的配置状态
	p.logger.WithFields(map[string]interface{}{
		"enable":  p.Config.Enable,
		"default": p.Config.Default,
	}).Info("Starting to save LLM config to file")

	// 保存LLM基本配置
	manager.Set("llm.enable", p.Config.Enable)
	manager.Set("llm.default", p.Config.Default)
	manager.Set("llm.provider", p.Config.Default)
	if sharedMode {
		manager.Set("llm.providers_file", p.Config.ProvidersFile)
		manager.Set("llm.model", p.Config.Model)
		p.writeSelectionKeys(manager)
	}

	// 旧模式才写凭据。共享模式下凭据的唯一来源是 providers 文件。
	if !sharedMode {
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
