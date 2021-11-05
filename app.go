package harukap

import (
	"github.com/allentom/haruka"
	"github.com/allentom/harukap/config"
	"github.com/allentom/harukap/youlog"
	"github.com/sirupsen/logrus"
)

type HarukaAppEngine struct {
	ConfigProvider *config.Provider
	Plugins        []HarukaPlugin
	LoggerPlugin   *youlog.Plugin
	HttpService    *haruka.Engine
}

func NewHarukaAppEngine() *HarukaAppEngine {
	return &HarukaAppEngine{
		Plugins: []HarukaPlugin{},
	}
}
func (e *HarukaAppEngine) UsePlugin(plugins ...HarukaPlugin) {
	e.Plugins = append(e.Plugins, plugins...)
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
	bootLogger.Info("init plugins")
	for _, plugin := range e.Plugins {
		err := plugin.OnInit(e)
		if err != nil {
			bootLogger.Fatal(err.Error())
		}
	}
	bootLogger.Info("start http service")
	e.HttpService.RunAndListen(e.ConfigProvider.Manager.GetString("addr"))
}
