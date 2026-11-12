package harukap

import (
	"fmt"
	"log"
	"net"

	"github.com/allentom/haruka"
	"github.com/allentom/harukap/config"
	"github.com/allentom/harukap/plugins/youlog"
	"github.com/allentom/harukap/rpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// LoggerPlugin 日志插件抽象，具体实现由 polaris toolkit
// （youplustoolkit/youlog.Plugin）提供。harukap 只依赖该接口，
// 不再引用 toolkit，保证依赖方向为 toolkit -> harukap -> haruka。
type LoggerPlugin interface {
	OnInit(*config.Provider) error
	NewScope(name string) *youlog.Scope
	Application() string
	Instance() string
}

type HarukaAppEngine struct {
	ConfigProvider       *config.Provider
	Plugins              []HarukaPlugin
	LoggerPlugin         LoggerPlugin
	HttpService          *haruka.Engine
	RPCService           *rpc.HarukaRPCService
	OnPluginInitComplete func()
}

func NewHarukaAppEngine() *HarukaAppEngine {
	return &HarukaAppEngine{
		Plugins: []HarukaPlugin{},
	}
}
func (e *HarukaAppEngine) UsePlugin(plugins ...HarukaPlugin) {
	e.Plugins = append(e.Plugins, plugins...)
}

// GetPluginsConfig 汇总所有实现了 PluginWithConfig 的插件配置
func (e *HarukaAppEngine) GetPluginsConfig() map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})
	for _, p := range e.Plugins {
		if pc, ok := p.(PluginWithConfig); ok {
			cfg := pc.GetPluginConfig()
			if cfg != nil {
				// 使用类型名作为键
				result[fmt.Sprintf("%T", p)] = cfg
			}
		}
	}
	return result
}
func (e *HarukaAppEngine) RunRPC() {
	lis, err := net.Listen("tcp", e.ConfigProvider.Manager.GetString("rpc.addr"))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	rpcServer := grpc.NewServer()
	e.RPCService.OnRegister(rpcServer)
	log.Printf("server listening at %v", lis.Addr())
	if err := rpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
func (e *HarukaAppEngine) Run() {
	if e.LoggerPlugin == nil {
		logrus.Fatal("LoggerPlugin is not set: assign a toolkit youlog.Plugin before Run()")
	}
	bootLogger := e.LoggerPlugin.NewScope("booting")
	bootLogger.WithFields(youlog.Fields{
		"Application": e.LoggerPlugin.Application(),
		"Instance":    e.LoggerPlugin.Instance(),
	}).Info("init logger success")
	bootLogger.Info("init plugins")
	for _, plugin := range e.Plugins {
		err := plugin.OnInit(e)
		if err != nil {
			bootLogger.Fatal(err.Error())
		}
	}

	if e.OnPluginInitComplete != nil {
		e.OnPluginInitComplete()
	}

	if e.RPCService != nil {
		bootLogger.Info("start rpc service")
		go e.RunRPC()
	}
	bootLogger.Info("start http service")
	e.HttpService.RunAndListen(e.ConfigProvider.Manager.GetString("addr"))
}
