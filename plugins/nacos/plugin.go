package nacos

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/allentom/harukap"
	"github.com/allentom/harukap/config"
	util "github.com/allentom/harukap/utils"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/sirupsen/logrus"
)

type NacosConfig struct {
	Enable      bool   `json:"enable"`
	Server      string `json:"server"`
	NamespaceId string `json:"namespaceId"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Group       string `json:"group"`
	ServiceName string `json:"serviceName"`
	ServiceIp   string `json:"serviceIp"`
}

type NacosPlugin struct {
	Logger       *logrus.Entry
	namingClient naming_client.INamingClient
	Config       *NacosConfig
	Port         int
}

func NewNacosPlugin(config *NacosConfig, port int) *NacosPlugin {
	return &NacosPlugin{
		Config: config,
		Port:   port,
	}
}

// NewNacosPluginFromYAML 通过 Provider 中的 yml 配置创建 Nacos 插件
// 读取键：
// nacos.enable, nacos.server, nacos.namespaceId, nacos.username, nacos.password,
// nacos.group, nacos.serviceName, nacos.serviceIp
// 端口默认从 addr 解析（格式 host:port），失败则使用 fallbackPort
func NewNacosPluginFromYAML(provider *config.Provider, defaultServiceName string, fallbackPort int) (*NacosPlugin, error) {
	if provider == nil || provider.Manager == nil {
		return nil, fmt.Errorf("nil config provider")
	}
	v := provider.Manager
	cfg := &NacosConfig{
		Enable:      v.GetBool("nacos.enable"),
		Server:      v.GetString("nacos.server"),
		NamespaceId: v.GetString("nacos.namespaceId"),
		Username:    v.GetString("nacos.username"),
		Password:    v.GetString("nacos.password"),
		Group:       v.GetString("nacos.group"),
		ServiceName: v.GetString("nacos.serviceName"),
		ServiceIp:   v.GetString("nacos.serviceIp"),
	}
	if cfg.Group == "" {
		cfg.Group = "DEFAULT_GROUP"
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = defaultServiceName
	}

	port := fallbackPort
	addr := v.GetString("addr")
	if addr != "" {
		_, portStr, err := net.SplitHostPort(addr)
		if err == nil {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}
	}

	return NewNacosPlugin(cfg, port), nil
}

func (p *NacosPlugin) OnInit(engine *harukap.HarukaAppEngine) error {
	p.Logger = logrus.New().WithFields(logrus.Fields{
		"scope": "Nacos Plugin",
	})

	if p.Config == nil || !p.Config.Enable {
		p.Logger.Info("nacos disabled, skip registration")
		return nil
	}

	// youlog config output
	yLogger := engine.LoggerPlugin.Logger.NewScope("NacosPlugin")
	yLogger.WithFields(map[string]interface{}{
		"enable":      p.Config.Enable,
		"server":      p.Config.Server,
		"namespaceId": p.Config.NamespaceId,
		"username":    p.Config.Username,
		"password":    util.MaskKeepHeadTail(p.Config.Password, 1, 2),
		"group":       p.Config.Group,
		"serviceName": p.Config.ServiceName,
		"serviceIp":   p.Config.ServiceIp,
		"port":        p.Port,
	}).Info("nacos config")

	serverHost := p.Config.Server
	serverIp := serverHost
	serverPort := uint64(8848)
	if idx := strings.LastIndex(serverHost, ":"); idx > 0 {
		serverIp = serverHost[:idx]
		var portParsed uint64
		_, err := fmt.Sscanf(serverHost[idx+1:], "%d", &portParsed)
		if err == nil && portParsed > 0 {
			serverPort = portParsed
		}
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: serverIp,
			Port:   serverPort,
		},
	}

	clientConfig := constant.ClientConfig{
		NamespaceId:         p.Config.NamespaceId,
		Username:            p.Config.Username,
		Password:            p.Config.Password,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogLevel:            "warn",
	}

	var err error
	p.namingClient, err = clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create nacos naming client: %v", err)
	}

	// 注册服务实例
	success, err := p.namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          p.Config.ServiceIp,
		Port:        uint64(p.Port),
		ServiceName: p.Config.ServiceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		GroupName:   p.Config.Group,
		ClusterName: "DEFAULT",
		Metadata:    map[string]string{},
	})
	if err != nil || !success {
		return fmt.Errorf("failed to register service instance: %v", err)
	}

	p.Logger.Info("service registered to nacos successfully")
	return nil
}

func (p *NacosPlugin) GetPluginConfig() map[string]interface{} {
	if p.Config == nil {
		return nil
	}
	return map[string]interface{}{
		"enable":      p.Config.Enable,
		"server":      p.Config.Server,
		"namespaceId": p.Config.NamespaceId,
		"username":    p.Config.Username,
		"password":    util.MaskKeepHeadTail(p.Config.Password, 1, 2),
		"group":       p.Config.Group,
		"serviceName": p.Config.ServiceName,
		"serviceIp":   p.Config.ServiceIp,
		"port":        p.Port,
	}
}

func (p *NacosPlugin) OnShutdown() error {
	if p.namingClient != nil {
		_, err := p.namingClient.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          p.Config.ServiceIp,
			Port:        uint64(p.Port),
			ServiceName: p.Config.ServiceName,
			GroupName:   p.Config.Group,
			Ephemeral:   true,
		})
		if err != nil {
			return fmt.Errorf("failed to deregister service instance: %v", err)
		}
		p.Logger.Info("service deregistered from nacos successfully")
	}
	return nil
}

// RegisterToNacos 使用当前 Nacos 客户端注册一个实例
func (p *NacosPlugin) RegisterToNacos(serviceIp string, port uint64, serviceName string, group string, metadata map[string]string) error {
	if p.namingClient == nil {
		return fmt.Errorf("nacos client not initialized")
	}
	if serviceIp == "" {
		serviceIp = p.Config.ServiceIp
	}
	if port == 0 {
		port = uint64(p.Port)
	}
	if serviceName == "" {
		serviceName = p.Config.ServiceName
	}
	if group == "" {
		group = p.Config.Group
	}
	if metadata == nil {
		metadata = map[string]string{}
	}

	ok, err := p.namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          serviceIp,
		Port:        port,
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		GroupName:   group,
		ClusterName: "DEFAULT",
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to register service instance: %v", err)
	}
	if !ok {
		return fmt.Errorf("failed to register service instance: result false")
	}
	return nil
}

// DeregisterFromNacos 使用当前 Nacos 客户端注销一个实例
func (p *NacosPlugin) DeregisterFromNacos(serviceIp string, port uint64, serviceName string, group string) error {
	if p.namingClient == nil {
		return fmt.Errorf("nacos client not initialized")
	}
	if serviceIp == "" {
		serviceIp = p.Config.ServiceIp
	}
	if port == 0 {
		port = uint64(p.Port)
	}
	if serviceName == "" {
		serviceName = p.Config.ServiceName
	}
	if group == "" {
		group = p.Config.Group
	}

	_, err := p.namingClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          serviceIp,
		Port:        port,
		ServiceName: serviceName,
		GroupName:   group,
		Ephemeral:   true,
	})
	if err != nil {
		return fmt.Errorf("failed to deregister service instance: %v", err)
	}
	return nil
}

func (p *NacosPlugin) OnStop() error {
	return p.OnShutdown()
}
