package cli

import (
	"github.com/allentom/harukap"
	"github.com/allentom/harukap/config"
	srv "github.com/kardianos/service"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"path/filepath"
)

type Wrapper struct {
	Config        *config.Provider
	ServiceConfig *srv.Config
	Service       AppService
}

func NewWrapper(engine *harukap.HarukaAppEngine) (*Wrapper, error) {
	w := &Wrapper{
		Config: engine.ConfigProvider,
	}
	err := w.InitService()
	if err != nil {
		return nil, err
	}
	w.Service = AppService{Program: func() {
		engine.Run()
	}}
	return w, err
}

func (w *Wrapper) InitService() error {
	workPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	w.ServiceConfig = &srv.Config{
		Name:             w.Config.Manager.GetString("service.name"),
		DisplayName:      w.Config.Manager.GetString("service.display"),
		WorkingDirectory: workPath,
		Arguments:        []string{"run"},
	}
	return nil
}

type AppService struct {
	Program func()
}

func (p *AppService) Start(s srv.Service) error {
	go p.Program()
	return nil
}

func (p *AppService) Stop(s srv.Service) error {
	return nil
}

func (w *Wrapper) InstallAsService() {
	prg := &AppService{}
	s, err := srv.New(prg, w.ServiceConfig)
	if err != nil {
		logrus.Fatal(err)
	}
	s.Uninstall()

	err = s.Install()
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("successful install service")
}

func (w *Wrapper) UnInstall() {

	prg := &AppService{}
	s, err := srv.New(prg, w.ServiceConfig)
	if err != nil {
		logrus.Fatal(err)
	}
	s.Uninstall()
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("successful uninstall service")
}

func (w *Wrapper) StartService() {
	prg := &AppService{}
	s, err := srv.New(prg, w.ServiceConfig)
	if err != nil {
		logrus.Fatal(err)
	}
	err = s.Start()
	if err != nil {
		logrus.Fatal(err)
	}
}
func (w *Wrapper) StopService() {
	prg := &AppService{}
	s, err := srv.New(prg, w.ServiceConfig)
	if err != nil {
		logrus.Fatal(err)
	}
	err = s.Stop()
	if err != nil {
		logrus.Fatal(err)
	}
}
func (w *Wrapper) RestartService() {
	prg := &AppService{}
	s, err := srv.New(prg, w.ServiceConfig)
	if err != nil {
		logrus.Fatal(err)
	}
	err = s.Restart()
	if err != nil {
		logrus.Fatal(err)
	}
}
func (w *Wrapper) RunApp() {
	app := &cli.App{
		Flags: []cli.Flag{},
		Commands: []*cli.Command{
			&cli.Command{
				Name:  "service",
				Usage: "service manager",
				Subcommands: []*cli.Command{
					{
						Name:  "install",
						Usage: "install service",
						Action: func(context *cli.Context) error {
							w.InstallAsService()
							return nil
						},
					},
					{
						Name:  "uninstall",
						Usage: "uninstall service",
						Action: func(context *cli.Context) error {
							w.UnInstall()
							return nil
						},
					},
					{
						Name:  "start",
						Usage: "start service",
						Action: func(context *cli.Context) error {
							w.StartService()
							return nil
						},
					},
					{
						Name:  "stop",
						Usage: "stop service",
						Action: func(context *cli.Context) error {
							w.StopService()
							return nil
						},
					},
					{
						Name:  "restart",
						Usage: "restart service",
						Action: func(context *cli.Context) error {
							w.RestartService()
							return nil
						},
					},
				},
				Description: "Service controller",
			},
			{
				Name:  "run",
				Usage: "run app",
				Action: func(context *cli.Context) error {
					w.Service.Program()
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
