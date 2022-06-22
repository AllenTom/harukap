package harukap

import (
	"github.com/allentom/haruka"
	"github.com/allentom/harukap/config"
	"github.com/allentom/harukap/plugins/youlog"
	"github.com/allentom/harukap/rpc"
	youlog2 "github.com/project-xpolaris/youplustoolkit/youlog"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"log"
	"net"
)

type HarukaAppEngine struct {
	ConfigProvider *config.Provider
	Plugins        []HarukaPlugin
	LoggerPlugin   *youlog.Plugin
	HttpService    *haruka.Engine
	RPCService     *rpc.HarukaRPCService
}

func NewHarukaAppEngine() *HarukaAppEngine {
	return &HarukaAppEngine{
		Plugins: []HarukaPlugin{},
	}
}
func (e *HarukaAppEngine) UsePlugin(plugins ...HarukaPlugin) {
	e.Plugins = append(e.Plugins, plugins...)
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
		e.LoggerPlugin = &youlog.Plugin{}
		err := e.LoggerPlugin.OnInit(e.ConfigProvider)
		if err != nil {
			logrus.Fatal(err)
		}
	}
	bootLogger := e.LoggerPlugin.Logger.NewScope("booting")
	bootLogger.WithFields(youlog2.Fields{
		"Application": e.LoggerPlugin.Logger.Application,
		"Instance":    e.LoggerPlugin.Logger.Instance,
	}).Info("init logger success")
	bootLogger.Info("init plugins")
	for _, plugin := range e.Plugins {
		err := plugin.OnInit(e)
		if err != nil {
			bootLogger.Fatal(err.Error())
		}
	}
	if e.RPCService != nil {
		bootLogger.Info("start rpc service")
		go e.RunRPC()
	}
	bootLogger.Info("start http service")
	e.HttpService.RunAndListen(e.ConfigProvider.Manager.GetString("addr"))
}
