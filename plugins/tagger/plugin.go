package tagger

import (
	"fmt"
	"io"

	"github.com/allentom/harukap"
)

// Config 是插件的配置结构体
type Config struct {
	Enable bool
	// 直接连接配置
	URL string
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	if !c.Enable {
		return nil
	}

	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	return nil
}

// 错误定义
var (
	ErrPluginDisabled       = fmt.Errorf("plugin is disabled")
	ErrClientNotInitialized = fmt.Errorf("client is not initialized")
	ErrConnectionFailed     = fmt.Errorf("connection failed")
)

// ImageTagger 是图片标签服务的核心接口
type ImageTagger interface {
	// TagImage 为图片添加标签
	TagImage(reader io.Reader, model string, threshold float64) ([]ImageTag, error)
	// GetInfo 获取服务信息
	GetInfo() (*ServiceInfo, error)
	// SwitchModel 切换模型
	SwitchModel(name string) error
	// GetTaggerState 获取标签器状态
	GetTaggerState() (*TaggerState, error)
}

// ImageTaggerPlugin 是图片标签插件
type ImageTaggerPlugin struct {
	client ImageTagger
	enable bool
	config *Config
}

// NewImageTaggerPlugin 创建一个新的图片标签插件
func NewImageTaggerPlugin() *ImageTaggerPlugin {
	return &ImageTaggerPlugin{}
}

// NewImageTaggerPluginWithConfig 使用配置创建一个新的图片标签插件
func NewImageTaggerPluginWithConfig(config *Config) (*ImageTaggerPlugin, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	plugin := &ImageTaggerPlugin{
		enable: config.Enable,
		config: config,
	}

	if !config.Enable {
		return plugin, nil
	}

	var err error
	plugin.client, err = initDirectClient(config)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	return plugin, nil
}

// initDirectClient 初始化直接连接客户端
func initDirectClient(config *Config) (ImageTagger, error) {
	client := NewClient(config.URL)

	// 检查连接
	info, err := client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get service info: %w", err)
	}
	if !info.Success {
		return nil, ErrConnectionFailed
	}

	return client, nil
}

// IsEnable 检查插件是否启用
func (i *ImageTaggerPlugin) IsEnable() bool {
	return i.enable && i.client != nil
}

// GetClient 获取标签器客户端
func (i *ImageTaggerPlugin) GetClient() (ImageTagger, error) {
	if !i.IsEnable() {
		return nil, ErrPluginDisabled
	}
	return i.client, nil
}

// TagImage 为图片添加标签
func (i *ImageTaggerPlugin) TagImage(reader io.Reader, model string, threshold float64) ([]ImageTag, error) {
	client, err := i.GetClient()
	if err != nil {
		return nil, err
	}
	return client.TagImage(reader, model, threshold)
}

// GetInfo 获取服务信息
func (i *ImageTaggerPlugin) GetInfo() (*ServiceInfo, error) {
	client, err := i.GetClient()
	if err != nil {
		return nil, err
	}
	return client.GetInfo()
}

// SwitchModel 切换模型
func (i *ImageTaggerPlugin) SwitchModel(name string) error {
	client, err := i.GetClient()
	if err != nil {
		return err
	}
	return client.SwitchModel(name)
}

// GetTaggerState 获取标签器状态
func (i *ImageTaggerPlugin) GetTaggerState() (*TaggerState, error) {
	client, err := i.GetClient()
	if err != nil {
		return nil, err
	}
	return client.GetTaggerState()
}

// OnInit 是 harukap 插件的初始化方法
func (i *ImageTaggerPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	logger := e.LoggerPlugin.Logger.NewScope("ImageTaggerPlugin")

	config := &Config{
		Enable: e.ConfigProvider.Manager.GetBool("imagetagger.enable"),
	}

	if !config.Enable {
		logger.Info("ImageTaggerPlugin is disabled")
		return nil
	}

	logger.Info("Init ImageTaggerPlugin")

	config.URL = e.ConfigProvider.Manager.GetString("imagetagger.url")
	logger.WithFields(map[string]interface{}{
		"enable": config.Enable,
		"url":    config.URL,
	}).Info("imagetagger config")

	// 使用新的配置初始化插件
	plugin, err := NewImageTaggerPluginWithConfig(config)
	if err != nil {
		logger.Error("Failed to initialize plugin: " + err.Error())
		return err
	}

	// 复制插件状态
	i.client = plugin.client
	i.enable = plugin.enable
	i.config = plugin.config

	return nil
}

func (i *ImageTaggerPlugin) GetPluginConfig() map[string]interface{} {
	if i.config == nil {
		return map[string]interface{}{"enable": i.enable}
	}
	cfg := map[string]interface{}{
		"enable": i.config.Enable,
	}
	cfg["mode"] = "direct"
	cfg["url"] = i.config.URL
	return cfg
}
